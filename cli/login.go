package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/k0kubun/pp"
	"github.com/redbubble/yak/cache"
	"github.com/redbubble/yak/okta"
	"github.com/redbubble/yak/saml"
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

func samlResponseCacheKey() string {
	return fmt.Sprintf("okta:samlResponse:%s:%s", viper.GetString("okta.domain"), viper.GetString("okta.username"))
}

func getSamlFromCache() (string, bool) {
	data, ok := cache.Check(samlResponseCacheKey()).(string)
	return data, ok
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
	samlPayload, gotSaml := getSamlFromCache()

	if !gotSaml {
		var authResponse okta.OktaAuthResponse
		var err error

		if viper.GetBool("cache.cache_only") {
			return saml.LoginData{}, errors.New("Could not find credentials in cache and --cache-only specified. Exiting.")
		}

		authResponse, err = promptLogin()

		if err != nil {
			return saml.LoginData{}, err

		}

		pp.Print(authResponse)

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

		pp.Print(authResponse)

		session, err := okta.CreateSession(viper.GetString("okta.domain"), authResponse)
		if err != nil {
			return saml.LoginData{}, err
		}

		samlPayload, err = okta.AwsSamlLogin(viper.GetString("okta.domain"), viper.GetString("okta.aws_saml_endpoint"), *session)
		if err != nil {
			return saml.LoginData{}, err
		}
	}

	samlResponse, err := saml.ParseResponse(samlPayload)

	if err != nil {
		return saml.LoginData{}, err
	}

	expiryTime := samlResponse.Assertion.Conditions.NotOnOrAfter

	cache.Write(samlResponseCacheKey(), string(samlPayload), expiryTime.Sub(time.Now()))
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
			fmt.Fprintf(os.Stderr, "Okta MFA token (from %s): ", okta.TotpFactorName(factor.Provider))
			passCode, _ := getLine()
			authResponse, err = okta.VerifyTotp(factor.Links.VerifyLink.Href, okta.TotpRequest{stateToken, passCode})
		case "token:hardware":
			fmt.Fprintf(os.Stderr, "Okta MFA token (from %s): ", okta.TotpFactorName(factor.Provider))
			passCode, _ := getLine()
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
		username := viper.GetString("okta.username")
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

			fmt.Fprintf(os.Stderr, "%s: ", prompt)
			password, err = getPassword()

			if err != nil {
				return authResponse, err
			}
		}

		authResponse, err = okta.Authenticate(viper.GetString("okta.domain"), okta.UserData{username, password})

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
