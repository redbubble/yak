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

const (
	YAK_STATUS_OK           = iota
	YAK_STATUS_UNAUTHORISED = iota
	YAK_STATUS_DATA_ERROR   = iota
	YAK_STATUS_NET_ERROR    = iota
	YAK_STATUS_BAD_RESPONSE = iota
)

type OktaAuthResponse struct {
	StateToken    string               `json:"stateToken"`
	SessionToken  string               `json:"sessionToken"`
	ExpiresAt     string               `json:"expiresAt"`
	Status        string               `json:"status"`
	Embedded      AuthResponseEmbedded `json:"_embedded"`
	YakStatusCode int
}

func Authenticate(oktaHref string, userData UserData) (OktaAuthResponse, error) {
	authBody, err := json.Marshal(userData)

	if err != nil {
		return OktaAuthResponse{YakStatusCode: YAK_STATUS_DATA_ERROR}, err
	}

	oktaUrl, err := url.Parse(oktaHref)

	if err != nil {
		return OktaAuthResponse{YakStatusCode: YAK_STATUS_DATA_ERROR}, err
	}

	primaryAuthEndpoint, _ := url.Parse("/api/v1/authn")
	primaryAuthUrl := oktaUrl.ResolveReference(primaryAuthEndpoint)

	resp, err := http.Post(primaryAuthUrl.String(), "application/json", bytes.NewBuffer(authBody))
	defer resp.Body.Close()

	if err != nil {
		return OktaAuthResponse{YakStatusCode: YAK_STATUS_NET_ERROR}, err
	} else if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return OktaAuthResponse{YakStatusCode: YAK_STATUS_UNAUTHORISED}, errors.New("Authentication failure (username or password invalid)")
	} else if resp.StatusCode >= 300 {
		return OktaAuthResponse{YakStatusCode: YAK_STATUS_NET_ERROR}, errors.New("Could not authenticate (" + resp.Status + ")")
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return OktaAuthResponse{YakStatusCode: YAK_STATUS_BAD_RESPONSE}, err
	}

	authResponse := OktaAuthResponse{YakStatusCode: YAK_STATUS_OK}
	json.Unmarshal(body, &authResponse)

	return authResponse, nil
}

func VerifyTotp(url string, totpRequestBody TotpRequest) (OktaAuthResponse, error) {
	totpJson, err := json.Marshal(totpRequestBody)

	if err != nil {
		return OktaAuthResponse{YakStatusCode: YAK_STATUS_DATA_ERROR}, err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(totpJson))
	defer resp.Body.Close()

	if err != nil {
		return OktaAuthResponse{YakStatusCode: YAK_STATUS_NET_ERROR}, err
	} else if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return OktaAuthResponse{YakStatusCode: YAK_STATUS_UNAUTHORISED}, errors.New("Authentication failure (MFA invalid)")
	} else if resp.StatusCode >= 300 {
		return OktaAuthResponse{YakStatusCode: YAK_STATUS_NET_ERROR}, errors.New("MFA failed (" + resp.Status + ")")
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return OktaAuthResponse{YakStatusCode: YAK_STATUS_BAD_RESPONSE}, err
	}

	authResponse := OktaAuthResponse{YakStatusCode: YAK_STATUS_OK}
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

	jar, err := cookiejar.New(nil)

	if err != nil {
		return "", err
	}

	client := http.Client{
		Jar: jar,
	}

	resp, err := client.Get(samlUrl.String())
	defer resp.Body.Close()

	if err != nil {
		return "", err
	} else if resp.StatusCode >= 300 {
		return "", errors.New("Could not get SAML payload" + resp.Status + ")")
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}

	data, err := extractSamlPayload(body)

	if err != nil {
		return "", err
	}

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

func TotpFactorName(key string) string {
	switch key {
	case "GOOGLE":
		return "Google Authenticator"
	default:
		return key
	}
}
