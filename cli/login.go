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

func GetLoginData() (saml.LoginData, error) {
	var samlPayload string

	if !viper.GetBool("cache.no_cache") {
		data, ok := cache.Check("okta:samlResponse").(string)

		if ok {
			fmt.Printf("Cache hit!")
			samlPayload = data
		}
	}

	if samlPayload == "" {
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
					fmt.Fprintf(os.Stderr, "MFA key (%s): ", factor.Provider)
					passCode, _ := getLine()

					authResponse, err = okta.VerifyTotp(factor.Links.VerifyLink.Href, okta.TotpRequest{authResponse.StateToken, passCode})
					break
				}
			}

			if err != nil {
				return saml.LoginData{}, err
			}
		}

		samlPayload, err := okta.AwsSamlLogin(viper.GetString("okta.domain"), viper.GetString("okta.aws_saml_endpoint"), authResponse)

		if err != nil {
			return saml.LoginData{}, err
		}

		if !viper.GetBool("cache.no_cache") {
			cache.Write("okta:samlResponse", string(samlPayload), 10*time.Minute)
		}
	}

	samlResponse, err := saml.ParseResponse(samlPayload)

	if err != nil {
		return saml.LoginData{}, err
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
