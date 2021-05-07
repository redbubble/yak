package okta

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
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

type PushRequest struct {
	StateToken string `json:"stateToken"`
}

type OktaLink struct {
	Href string `json:"href"`
}

type AuthResponseFactorLinks struct {
	VerifyLink OktaLink `json:"verify"`
}

type PushRequestResponseLinks struct {
	PollLink OktaLink `json:"next"`
}

type PushRequestResponse struct {
	Links        PushRequestResponseLinks `json:"_links"`
	FactorResult string                   `json:"factorResult"`
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

type OktaSession struct {
	Id        string    `json:"id"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// TODO: DRY
func CreateSession(oktaHref string, authResponse OktaAuthResponse) (*OktaSession, error) {
	authBody, err := json.Marshal(map[string]string{"sessionToken": authResponse.SessionToken})
	if err != nil {
		return nil, err
	}

	oktaUrl, err := url.Parse(oktaHref)
	if err != nil {
		return nil, err
	}

	sessionEndpoint, _ := url.Parse("/api/v1/sessions")
	sessionUrl := oktaUrl.ResolveReference(sessionEndpoint)

	resp, err := http.Post(sessionUrl.String(), "application/json", bytes.NewBuffer(authBody))
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	session := OktaSession{}
	if err := json.Unmarshal(body, &session); err != nil {
		return nil, err
	}
	log.WithField("session", session).Debug("okta.go: Created Session from Okta")
	return &session, nil
}

// TODO: DRY
func GetSession(oktaHref string, session *OktaSession) (*OktaSession, error) {
	oktaUrl, err := url.Parse(oktaHref)
	if err != nil {
		return nil, err
	}

	sessionEndpoint, _ := url.Parse("/api/v1/sessions/me")
	sessionUrl := oktaUrl.ResolveReference(sessionEndpoint)

	jar, _ := cookiejar.New(nil)
	jar.SetCookies(sessionUrl, []*http.Cookie{{Name: "sid", Value: session.Id}})

	client := http.Client{
		Jar: jar,
	}

	resp, err := client.Get(sessionUrl.String())
	if err != nil {
		return nil, err
	}

	if resp.StatusCode > 300 {
		return nil, fmt.Errorf("Status code %d, expected < 2xx", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	newSession := OktaSession{}
	if err := json.Unmarshal(body, &newSession); err != nil {
		return nil, err
	}
	log.WithField("session", string(body)).Debug("okta.go: Retrieved Session from Okta")
	return &newSession, nil
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

	body, yakStatus, err := makeRequest(primaryAuthUrl.String(), bytes.NewBuffer(authBody))

	if err != nil {
		log.WithField("err", err).Debug("okta.go: Okta login error")
		return OktaAuthResponse{YakStatusCode: yakStatus}, err
	}

	authResponse := OktaAuthResponse{YakStatusCode: YAK_STATUS_OK}
	json.Unmarshal(body, &authResponse)
	log.WithField("response", authResponse).Debug("okta.go: Auth response for Okta login")

	return authResponse, nil
}

func VerifyTotp(url string, totpRequestBody TotpRequest) (OktaAuthResponse, error) {
	totpJson, err := json.Marshal(totpRequestBody)

	if err != nil {
		return OktaAuthResponse{YakStatusCode: YAK_STATUS_DATA_ERROR}, err
	}

	body, yakStatus, err := makeRequest(url, bytes.NewBuffer(totpJson))

	if err != nil {
		return OktaAuthResponse{YakStatusCode: yakStatus}, err
	}

	authResponse := OktaAuthResponse{YakStatusCode: YAK_STATUS_OK}
	json.Unmarshal(body, &authResponse)

	return authResponse, nil
}

func VerifyPush(url string, pushRequestBody PushRequest) (OktaAuthResponse, error) {
	pushJson, err := json.Marshal(pushRequestBody)

	if err != nil {
		return OktaAuthResponse{YakStatusCode: YAK_STATUS_DATA_ERROR}, err
	}

	body, yakStatus, err := makeRequest(url, bytes.NewBuffer(pushJson))

	if err != nil {
		return OktaAuthResponse{YakStatusCode: yakStatus}, err
	}

	pushRequestResponse := PushRequestResponse{}
	json.Unmarshal(body, &pushRequestResponse)

	errorsRemaining := 6
	fmt.Fprintf(os.Stderr, "Waiting for MFA response")
	for {
		body, yakStatus, err := makeRequest(pushRequestResponse.Links.PollLink.Href, bytes.NewBuffer(pushJson))

		authResponse := OktaAuthResponse{YakStatusCode: yakStatus}

		if err != nil {
			errorsRemaining--
			if errorsRemaining == 0 {
				fmt.Fprintf(os.Stderr, "\nToo many network errors, aborting...")
				return authResponse, err
			}
			continue
		}

		json.Unmarshal(body, &authResponse)

		if authResponse.Status != "MFA_CHALLENGE" {
			if authResponse.Status == "SUCCESS" {
				fmt.Fprintf(os.Stderr, "\n")
				return authResponse, nil
			}

			fmt.Fprintf(os.Stderr, "\n")
			authResponse.YakStatusCode = YAK_STATUS_BAD_RESPONSE
			return authResponse, errors.New("Bad status from Okta API: " + authResponse.Status)
		}

		fmt.Fprintf(os.Stderr, ".")

		time.Sleep(5 * time.Second)
	}
}

func AwsSamlLogin(oktaHref string, samlHref string, oktasession OktaSession) (string, error) {
	oktaUrl, err := url.Parse(oktaHref)

	if err != nil {
		return "", err
	}

	samlEndpoint, err := url.Parse(samlHref)

	if err != nil {
		return "", err
	}

	samlUrl := oktaUrl.ResolveReference(samlEndpoint)

	jar, err := cookiejar.New(nil)
	jar.SetCookies(samlUrl, []*http.Cookie{{Name: "sid", Value: oktasession.Id}})

	if err != nil {
		return "", err
	}

	client := http.Client{
		Jar: jar,
	}

	resp, err := client.Get(samlUrl.String())

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

func makeRequest(url string, body io.Reader) ([]byte, int, error) {
	resp, err := http.Post(url, "application/json", body)
	log.WithField("url", url).WithField("statusCode", resp.StatusCode).Debug("okta.go: Okta request")

	if err != nil {
		return []byte{}, YAK_STATUS_NET_ERROR, err
	} else if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return []byte{}, YAK_STATUS_UNAUTHORISED, errors.New("Unauthorised (" + resp.Status + ")")
	} else if resp.StatusCode >= 300 {
		return []byte{}, YAK_STATUS_NET_ERROR, errors.New("Network error (" + resp.Status + ")")
	}

	defer resp.Body.Close()

	responseBody, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return responseBody, YAK_STATUS_BAD_RESPONSE, err
	}

	return responseBody, YAK_STATUS_OK, err
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
