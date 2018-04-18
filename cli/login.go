package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/redbubble/yak/okta"
	"github.com/redbubble/yak/saml"
)

func GetLoginData() (saml.LoginData, error) {
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

	samlResponse, err := saml.ParseResponse(samlPayload)

	if err != nil {
		return saml.LoginData{}, err
	}

	return saml.CreateLoginData(samlResponse, samlPayload), nil
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
