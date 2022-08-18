package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/viper"
	"github.com/twpayne/go-pinentry"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/redbubble/yak/cache"
	"github.com/redbubble/yak/okta"
	"github.com/redbubble/yak/saml"
	log "github.com/sirupsen/logrus"
)

const maxLoginRetries = 3

var acceptableAuthFactors = [...]string{
	"token:software:totp",
	"token:hardware",
	"push",
}

func GetRolesFromCache() ([]saml.LoginRole, bool) {
	data, ok := cache.Check("aws:roles").([]string)

	if !ok {
		return []saml.LoginRole{}, false
	}

	roles := []saml.LoginRole{}
	for _, datum := range data {
		role, ok := saml.CreateLoginRole(datum)

		if ok {
			roles = append(roles, role)
		}
	}

	return roles, true
}

func oktaDomain() string {
	return viper.GetString("okta.domain")
}

func oktaUsername() string {
	return viper.GetString("okta.username")
}

func oktaSessionCacheKey() string {
	return fmt.Sprintf("okta:sessionToken:%s:%s", oktaDomain(), oktaUsername())
}

func getOktaSessionFromCache() (*okta.OktaSession, bool) {
	data, ok := cache.Check(oktaSessionCacheKey()).(okta.OktaSession)
	return &data, ok
}

func cacheOktaSession(session *okta.OktaSession) {
	expires := session.ExpiresAt.Sub(time.Now())
	expiryLimit := time.Duration(viper.GetInt64("okta.session_cache_limit")) * time.Second

	if expiryLimit > 0 && expiryLimit < expires {
		log.Debugf("Okta session expires in %.0f seconds, but we're configured to only cache that for %.0f seconds", expires.Seconds(), expiryLimit.Seconds())
		expires = expiryLimit
	}

	cache.Write(oktaSessionCacheKey(), *session, expires)
}

func checkOktaSession(session *okta.OktaSession) bool {
	response, err := okta.GetSession(oktaDomain(), session)

	// This needs explaining: Okta's "Create Session" API call gives
	// us a session ID that we set as the `sid` cookie. Get & Refresh return a
	// *different* ID that we can't use as the cookie, but they both
	// extend the calling session.

	if err == nil {
		session.ExpiresAt = response.ExpiresAt
		cacheOktaSession(session)
	}

	return err == nil
}

func GetLoginDataWithTimeout() (saml.LoginData, error) {
	errorChannel := make(chan error)
	resultChannel := make(chan saml.LoginData)

	go func() {
		data, err := getLoginData()

		if err != nil {
			errorChannel <- err
		} else {
			resultChannel <- data
		}
	}()

	timeoutSeconds := viper.GetDuration("login.timeout") * time.Second

	if timeoutSeconds != 0 {
		select {
		case err := <-errorChannel:
			return saml.LoginData{}, err
		case data := <-resultChannel:
			return data, nil
		case <-time.After(timeoutSeconds):
			return saml.LoginData{}, errors.New("Login timeout")
		}
	} else {
		select {
		case err := <-errorChannel:
			return saml.LoginData{}, err
		case data := <-resultChannel:
			return data, nil
		}
	}
}

func getLoginData() (saml.LoginData, error) {
	session, gotSession := getOktaSessionFromCache()

	if gotSession && session.ExpiresAt.After(time.Now()) {
		log.Infof("Okta session found in cache (%s), expires %s", session.Id, session.ExpiresAt.String())
		gotSession = checkOktaSession(session)
		if gotSession {
			log.Infof("Refreshed session, now expires %s", session.ExpiresAt.String())
		}
	}

	if !gotSession {
		var authResponse okta.OktaAuthResponse
		var err error

		log.Infof("Okta session not in cache or no longer valid, re-authenticating")

		if viper.GetBool("cache.cache_only") {
			return saml.LoginData{}, errors.New("Could not find credentials in cache and --cache-only specified. Run `yak <role>` to remedy.")
		}

		authResponse, err = promptLogin()

		if err != nil {
			return saml.LoginData{}, err

		}

		for authResponse.Status == "MFA_REQUIRED" {
			selectedFactor, err := chooseMFA(authResponse)

			if err != nil {
				return saml.LoginData{}, err
			}

			authResponse, err = promptMFA(selectedFactor, authResponse.StateToken)

			if err != nil {
				return saml.LoginData{}, err
			}
		}

		session, err = getOktaSession(authResponse)
		if err != nil {
			return saml.LoginData{}, err
		}

	}

	samlPayload, err := okta.AwsSamlLogin(oktaDomain(), viper.GetString("okta.aws_saml_endpoint"), *session)
	if err != nil {
		return saml.LoginData{}, err
	}

	samlResponse, err := saml.ParseResponse(samlPayload)

	if err != nil {
		return saml.LoginData{}, err
	}
	log.WithField("saml", samlResponse).Debug("okta.go: SAML response from Okta")

	return saml.CreateLoginData(samlResponse, samlPayload), nil
}

func chooseMFA(authResponse okta.OktaAuthResponse) (okta.AuthResponseFactor, error) {
	acceptableFactors := getAcceptableFactors(authResponse.Embedded.Factors)

	if len(acceptableFactors) == 0 {
		return okta.AuthResponseFactor{}, errors.New("No usable MFA factors found, but MFA was requested. Aborting.")
	}

	factor, gotFactor := getConfiguredMFAFactor(acceptableFactors)

	if gotFactor {
		return factor, nil
	} else if len(acceptableFactors) > 1 {
		var factorIndex int
		validIndex := false
		maxIndex := len(acceptableFactors) - 1

		for index, factor := range acceptableFactors {
			fmt.Fprintf(os.Stderr, "[%d] %s (%s)\n", index, factor.FactorType, factor.Provider)
		}

		for !validIndex {
			fmt.Fprint(os.Stderr, "Select an MFA factor (0): ")
			factorIndexString, err := getLine()

			if err != nil {
				return factor, err
			}

			if factorIndexString == "" {
				factorIndex = 0
			} else {
				factorIndex, err = strconv.Atoi(factorIndexString)

				if err != nil || factorIndex > maxIndex || factorIndex < 0 {
					fmt.Fprintf(os.Stderr, "Please enter a number between 0 and %d\n", maxIndex)
					continue
				}
			}

			factor = acceptableFactors[factorIndex]
			validIndex = true
		}

		fmt.Fprintf(os.Stderr, "Set as default MFA factor by adding mfa_type = \"%s\" and mfa_provider = \"%s\" to the [okta] section in your config!\n", factor.FactorType, factor.Provider)
		return factor, nil
	}

	// If no factor is chosen by this point, take the first acceptable factor
	return acceptableFactors[0], nil
}

func getOktaSession(authResponse okta.OktaAuthResponse) (session *okta.OktaSession, err error) {
	log.Infof("Creating new Okta session for %s", oktaDomain())
	session, err = okta.CreateSession(oktaDomain(), authResponse)

	if err == nil {
		cacheOktaSession(session)
	}

	return
}

func getAcceptableFactors(factors []okta.AuthResponseFactor) []okta.AuthResponseFactor {
	acceptableFactors := []okta.AuthResponseFactor{}

	for _, factor := range factors {
		if factorAcceptable(factor) {
			acceptableFactors = append(acceptableFactors, factor)
		}
	}

	return acceptableFactors
}

func factorAcceptable(factor okta.AuthResponseFactor) bool {
	for _, acceptableFactor := range acceptableAuthFactors {
		if factor.FactorType == acceptableFactor {
			return true
		}
	}

	return false
}

func getConfiguredMFAFactor(factors []okta.AuthResponseFactor) (okta.AuthResponseFactor, bool) {
	providerAcceptable := false
	typeAcceptable := false

	if viper.GetString("okta.mfa_type") != "" || viper.GetString("okta.mfa_provider") != "" {
		for _, factor := range factors {
			if factor.FactorType == viper.GetString("okta.mfa_type") {
				typeAcceptable = true

				if factor.Provider == strings.ToUpper(viper.GetString("okta.mfa_provider")) {
					providerAcceptable = true
					return factor, true
				}
			}
		}

		if !typeAcceptable {
			fmt.Fprintf(os.Stderr, "Warning: no factors of type '%s' available\n", viper.GetString("okta.mfa_type"))
		} else if !providerAcceptable {
			fmt.Fprintf(os.Stderr, "Warning: no factors from provider %s available\n", viper.GetString("okta.mfa_provider"))
		}
	}

	return okta.AuthResponseFactor{}, false
}

func promptMFA(factor okta.AuthResponseFactor, stateToken string) (okta.OktaAuthResponse, error) {
	var authResponse okta.OktaAuthResponse
	var err error
	retries := 0
	unauthorised := true

	for unauthorised && (retries < maxLoginRetries) {
		retries++

		switch factor.FactorType {
		case "push":
			authResponse, err = okta.VerifyPush(factor.Links.VerifyLink.Href, okta.PushRequest{stateToken})
		case "token:software:totp":
			passCode, _ := promptOrPinentry(fmt.Sprintf("Okta MFA token (from %s): ", okta.TotpFactorName(factor.Provider)), false)
			authResponse, err = okta.VerifyTotp(factor.Links.VerifyLink.Href, okta.TotpRequest{stateToken, passCode})
		case "token:hardware":
			passCode, _ := promptOrPinentry(fmt.Sprintf("Okta MFA token (from %s): ", okta.TotpFactorName(factor.Provider)), false)
			authResponse, err = okta.VerifyTotp(factor.Links.VerifyLink.Href, okta.TotpRequest{stateToken, passCode})
		default:
			err := errors.New("Unknown factor type selected. Exiting.")
			return authResponse, err
		}

		if authResponse.YakStatusCode == okta.YAK_STATUS_UNAUTHORISED && retries < maxLoginRetries {
			fmt.Fprintln(os.Stderr, "Sorry, Try again.")
		} else {
			unauthorised = false
		}
	}

	return authResponse, err
}

func promptLogin() (okta.OktaAuthResponse, error) {
	var authResponse okta.OktaAuthResponse
	var err error
	retries := 0
	unauthorised := true

	for unauthorised && (retries < maxLoginRetries) {
		retries++
		username := oktaUsername()
		promptUsername := (username == "")

		// Viper isn't used here because it's really hard to get Viper to not accept values through the config file
		password := os.Getenv("OKTA_PASSWORD")
		envPassword := (password != "")

		if promptUsername {
			fmt.Fprint(os.Stderr, "Okta username: ")
			username, err = getLine()

			if err != nil {
				return authResponse, err
			}
		}

		if password == "" {
			prompt := "Okta password"
			if !promptUsername {
				prompt = prompt + " (" + username + ")"
			}

			password, err = promptOrPinentry(fmt.Sprintf("%s: ", prompt), true)

			if err != nil {
				return authResponse, err
			}
		}

		authResponse, err = okta.Authenticate(oktaDomain(), okta.UserData{username, password})

		if authResponse.YakStatusCode == okta.YAK_STATUS_UNAUTHORISED && retries < maxLoginRetries && !envPassword {
			fmt.Fprintln(os.Stderr, "Sorry, try again.")
		} else {
			unauthorised = false
		}
	}

	return authResponse, err
}

func CacheLoginRoles(roles []saml.LoginRole) {
	data := []string{}

	for _, role := range roles {
		data = append(data, saml.SerialiseLoginRole(role))
	}

	cache.WriteDefault("aws:roles", data)
}

func promptOrPinentry(prompt string, secret bool) (string, error) {
	// Whether to use pinentry for (GUI) password prompt, or the original way
	if viper.GetBool("pinentry") {
		return getPinentry(prompt, secret)
	} else {
		fmt.Fprintf(os.Stderr, prompt)
		// If it's secret, don't echo the user's response.
		if secret {
			return getPassword()
		} else {
			return getLine()
		}
	}
}

func getPinentry(prompt string, secret bool) (string, error) {
	var p *pinentry.Client
	var err error

	if runtime.GOOS == "darwin" {
		p, err = pinentry.NewClient(pinentry.WithBinaryName("pinentry-mac"),
			pinentry.WithDesc(prompt),
			pinentry.WithPrompt(""),
			pinentry.WithTitle("Yak"))
	} else {
		p, err = pinentry.NewClient(pinentry.WithBinaryNameFromGnuPGAgentConf(),
			pinentry.WithDesc(prompt),
			pinentry.WithPrompt(""),
			pinentry.WithTitle("Yak"))
	}

	if err != nil {
		return "", fmt.Errorf("pinentry error: %w", err)
	}
	defer p.Close()

	pw, _, err := p.GetPIN()
	if err != nil {
		return "", fmt.Errorf("pinentry error: %w", err)
	}

	pass := string(pw)
	return pass, nil
}

func getPassword() (string, error) {
	bytes, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Fprint(os.Stderr, "\n")

	return string(bytes), err
}

func getLine() (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

	return scanner.Text(), scanner.Err()
}
