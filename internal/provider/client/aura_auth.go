package client

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type AuraAuth struct {
	clientId     string
	clientSecret string

	mutex *sync.Mutex
	token *TokenResponse

	httpClient *http.Client
}

type AuraAuthToken struct {
	token      string
	expiringAt int64
}

func (a *AuraAuth) authenticate() error {
	authUrl, err := url.Parse(auraBasePath + "/oauth/token")
	if err != nil {
		return err
	}

	var encodedCreds []byte
	base64.StdEncoding.Encode(encodedCreds, []byte(a.clientId+":"+a.clientSecret))

	req := &http.Request{
		Method: "POST",
		URL:    authUrl,
		Header: map[string][]string{
			"Content-Type":  {"application/x-www-form-urlencoded"},
			"Authorization": {"Basic " + string(encodedCreds)},
		},
		Body: io.NopCloser(strings.NewReader("grant_type=client_credentials")),
	}

	// todo retry
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	// todo check response status
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var token TokenResponse
	err = json.Unmarshal(body, &token)
	if err != nil {
		return err
	}

	a.token = &token

	return nil
}

func (a *AuraAuth) GetToken() (string, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if a.token == nil || a.token.ExpiredIn < time.Now().Unix()+60 {
		err := a.authenticate()
		if err == nil {
			return "", err
		}
	}
	return a.token.AccessToken, nil
}
