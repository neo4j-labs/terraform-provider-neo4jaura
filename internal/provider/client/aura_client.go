package client

import (
	"io"
	"net/http"
	"net/url"
)

const (
	auraBasePath = "https://api.neo4j.io"
	auraV1Path   = auraBasePath + "/v1"
)

type AuraClient struct {
	auth       *AuraAuth
	httpClient *http.Client
}

func NewAuraClient(clientId, clientSecret string) *AuraClient {
	httpClient := http.DefaultClient
	return &AuraClient{
		auth: &AuraAuth{
			clientId:     clientId,
			clientSecret: clientSecret,
			httpClient:   httpClient,
		},
		httpClient: httpClient,
	}
}

func (c *AuraClient) Get(path string) ([]byte, int, error) {
	token, err := c.auth.GetToken()
	if err != nil {
		return []byte{}, 0, err
	}

	getUrl, err := url.Parse(auraV1Path + "/" + path)
	if err != nil {
		return []byte{}, 0, err
	}

	req := &http.Request{
		Method: "GET",
		URL:    getUrl,
		Header: map[string][]string{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer " + token},
		},
	}

	// todo retry
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return []byte{}, 0, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, 0, err
	}
	return body, resp.StatusCode, nil
}
