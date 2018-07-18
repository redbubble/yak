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
	if viper.GetBool("cache.no_cache") {
		return []saml.LoginRole{}, false
	}

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
	if !viper.GetBool("cache.no_cache") {
		data, ok := cache.Check(samlResponseCacheKey()).(string)

		return data, ok
	}

	return "", false
}

func GetLoginData() (saml.LoginData, error) {
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

		samlPayload, err = okta.AwsSamlLogin(viper.GetString("okta.domain"), viper.GetString("okta.aws_saml_endpoint"), authResponse)

		if err != nil {
			return saml.LoginData{}, err
		}
	}

	samlResponse, err := saml.ParseResponse(samlPayload)

	if err != nil {
		return saml.LoginData{}, err
	}

	expiryTime := samlResponse.Assertion.Conditions.NotOnOrAfter

	if !viper.GetBool("cache.no_cache") {
		cache.Write(samlResponseCacheKey(), string(samlPayload), expiryTime.Sub(time.Now()))
	}

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
		for index, factor := range acceptableFactors {
			fmt.Fprintf(os.Stderr, "[%d] %s (%s)\n", index, factor.FactorType, factor.Provider)
		}

		fmt.Fprint(os.Stderr, "Select an MFA factor (0): ")
		factorIndexString, _ := getLine()

		if factorIndexString != "" {
			factorIndex, _ := strconv.Atoi(factorIndexString)
			factor = acceptableFactors[factorIndex]

			fmt.Fprintf(os.Stderr, "Set as default MFA factor by adding mfa_type = %s and mfa_provider = %s in your config!\n", factor.FactorType, factor.Provider)

			return factor, nil
		}
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
			print(factor.FactorType)
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

		if username == "" {
			fmt.Fprint(os.Stderr, "Okta username: ")
			username, _ = getLine()
		}

		fmt.Fprint(os.Stderr, "Okta password: ")
		password, _ := getPassword()

		authResponse, err = okta.Authenticate(viper.GetString("okta.domain"), okta.UserData{username, password})

		if authResponse.YakStatusCode == okta.YAK_STATUS_UNAUTHORISED && retries < maxLoginRetries {
			fmt.Fprintln(os.Stderr, "Sorry, try again.")
		} else {
			unauthorised = false
		}
	}

	return authResponse, err
}

func CacheLoginRoles(roles []saml.LoginRole) {
	if viper.GetBool("cache.no_cache") {
		return
	}

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
