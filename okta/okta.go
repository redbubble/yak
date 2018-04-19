package okta

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"golang.org/x/net/html"
)

type UserData struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type TotpRequest struct {
	StateToken string `json:"stateToken"`
	PassCode   string `json:"passCode"`
}

type OktaLink struct {
	Href string `json:"href"`
}

type AuthResponseFactorLinks struct {
	VerifyLink OktaLink `json:"verify"`
}

type AuthResponseFactor struct {
	Links      AuthResponseFactorLinks `json:"_links"`
	FactorType string                  `json:"factorType"`
	Provider   string                  `json:"provider"`
}

type AuthResponseEmbedded struct {
	Factors []AuthResponseFactor `json:"factors"`
}

type OktaAuthResponse struct {
	StateToken   string               `json:"stateToken"`
	SessionToken string               `json:"sessionToken"`
	ExpiresAt    string               `json:"expiresAt"`
	Status       string               `json:"status"`
	Embedded     AuthResponseEmbedded `json:"_embedded"`
}

func Authenticate(oktaHref string, userData UserData) (OktaAuthResponse, error) {
	authBody, err := json.Marshal(userData)

	if err != nil {
		return OktaAuthResponse{}, err
	}

	oktaUrl, err := url.Parse(oktaHref)

	if err != nil {
		return OktaAuthResponse{}, err
	}

	primaryAuthEndpoint, _ := url.Parse("/api/v1/authn")
	primaryAuthUrl := oktaUrl.ResolveReference(primaryAuthEndpoint)

	resp, err := http.Post(primaryAuthUrl.String(), "application/json", bytes.NewBuffer(authBody))
	defer resp.Body.Close()

	if err != nil {
		return OktaAuthResponse{}, err
	} else if resp.StatusCode >= 300 {
		return OktaAuthResponse{}, errors.New("Could not authenticate (" + resp.Status + ")")
	}

	body, _ := ioutil.ReadAll(resp.Body)

	var authResponse OktaAuthResponse
	json.Unmarshal(body, &authResponse)

	return authResponse, nil
}

func VerifyTotp(url string, totpRequestBody TotpRequest) (OktaAuthResponse, error) {
	totpJson, err := json.Marshal(totpRequestBody)

	if err != nil {
		return OktaAuthResponse{}, err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(totpJson))
	defer resp.Body.Close()

	if err != nil {
		return OktaAuthResponse{}, err
	} else if resp.StatusCode >= 300 {
		return OktaAuthResponse{}, errors.New("MFA failed (" + resp.Status + ")")
	}

	body, _ := ioutil.ReadAll(resp.Body)

	var authResponse OktaAuthResponse
	json.Unmarshal(body, &authResponse)

	return authResponse, nil
}

func AwsSamlLogin(oktaHref string, samlHref string, oktaAuthResponse OktaAuthResponse) (string, error) {
	oktaUrl, err := url.Parse(oktaHref)

	if err != nil {
		return "", err
	}

	samlEndpoint, err := url.Parse(samlHref)

	if err != nil {
		return "", err
	}

	samlUrl := oktaUrl.ResolveReference(samlEndpoint)

	query := url.Values{}
	query.Add("onetimetoken", oktaAuthResponse.SessionToken)

	samlUrl.RawQuery = query.Encode()

	jar, _ := cookiejar.New(nil)

	client := http.Client {
		Jar: jar,
	}

 	resp, _ := client.Get(samlUrl.String())
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	data, _ := extractSamlPayload(body)

	saml, err := base64.StdEncoding.DecodeString(data)

	if err != nil {
		return "", err
	}

	return string(saml), nil
}

func extractSamlPayload(htmlDocument []byte) (string, error) {
	tokeniser := html.NewTokenizer(bytes.NewBuffer(htmlDocument))

	var data string

	for {
		tokeniser.Next()
		token := tokeniser.Token()

		if token.Type == html.ErrorToken {
			return "", errors.New("No SAML payload found in response from Okta")
		}

		if (token.Type == html.SelfClosingTagToken || token.Type == html.StartTagToken) && token.Data == "input" {
			var inputName string
			var inputValue string

			for _, attribute := range token.Attr {
				if attribute.Key == "name" {
					inputName = attribute.Val
				}

				if attribute.Key == "value" {
					inputValue = attribute.Val
				}
			}

			if inputName == "SAMLResponse" {
				data = inputValue
				break
			}
		}
	}

	return data, nil
}
