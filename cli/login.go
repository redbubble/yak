package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/redbubble/yak/cache"
	"github.com/redbubble/yak/okta"
	"github.com/redbubble/yak/saml"
)

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
		if viper.GetBool("cache.cache_only") {
			return saml.LoginData{}, errors.New("Could not find credentials in cache and --cache-only specified. Exiting.")
		}

		username := viper.GetString("okta.username")

		if username == "" {
			fmt.Fprint(os.Stderr, "username: ")
			username, _ = getLine()
		}

		fmt.Fprint(os.Stderr, "password: ")
		password, _ := getPassword()

		authResponse, err := okta.Authenticate(viper.GetString("okta.domain"), okta.UserData{username, password})

		if err != nil {
			return saml.LoginData{}, err
		}

		for authResponse.Status == "MFA_REQUIRED" {
			for _, factor := range authResponse.Embedded.Factors {
				if factor.FactorType == "token:software:totp" {
					fmt.Fprintf(os.Stderr, "Okta MFA token (from %s): ", okta.TotpFactorName(factor.Provider))
					passCode, _ := getLine()

					authResponse, err = okta.VerifyTotp(factor.Links.VerifyLink.Href, okta.TotpRequest{authResponse.StateToken, passCode})
					break
				}
			}

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
	reader := bufio.NewReader(os.Stdin)
	username, err := reader.ReadString('\n')
	username = strings.Replace(username, "\n", "", -1)

	return username, err
}
