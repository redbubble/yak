package saml

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

type samlResponse struct {
	Assertion samlAssertion `xml:"Assertion"`
}

type samlAssertion struct {
	Attributes []samlAssertionAttribute `xml:"AttributeStatement>Attribute"`
	Conditions samlAssertionConditions  `xml:Conditions`
}

type samlAssertionConditions struct {
	NotBefore    time.Time `xml:"NotBefore,attr"`
	NotOnOrAfter time.Time `xml:"NotOnOrAfter,attr"`
}

type samlAssertionAttribute struct {
	Name   string   `xml:"Name,attr"`
	Values []string `xml:"AttributeValue"`
}

type LoginRole struct {
	RoleArn      string
	PrincipalArn string
}

type LoginData struct {
	Roles     []LoginRole
	Assertion string
}

func ParseResponse(saml string) (samlResponse, error) {
	var response samlResponse
	err := xml.Unmarshal([]byte(saml), &response)

	return response, err
}

func CreateLoginData(response samlResponse, payload string) LoginData {
	login := LoginData{
		Roles:     []LoginRole{},
		Assertion: base64.StdEncoding.EncodeToString([]byte(payload)),
	}

	for _, attribute := range response.Assertion.Attributes {
		if attribute.Name == "https://aws.amazon.com/SAML/Attributes/Role" {
			for _, value := range attribute.Values {
				role, ok := CreateLoginRole(value)

				if ok {
					login.Roles = append(login.Roles, role)
				}
			}
		}
	}

	return login
}

func CreateLoginRole(roleData string) (LoginRole, bool) {
	parts := strings.Split(roleData, ",")

	if len(parts) == 2 {
		return LoginRole{
			RoleArn:      parts[1],
			PrincipalArn: parts[0],
		}, true
	} else {
		return LoginRole{}, false
	}
}

func SerialiseLoginRole(role LoginRole) string {
	return fmt.Sprintf("%s,%s", role.PrincipalArn, role.RoleArn)
}

func (login LoginData) GetLoginRole(roleArn string) (LoginRole, error) {
	for _, role := range login.Roles {
		if role.RoleArn == roleArn {
			return role, nil
		}
	}

	return LoginRole{}, fmt.Errorf("ARN %s is not in the list of available roles for this user", roleArn)
}
