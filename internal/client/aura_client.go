/*
 *  Copyright (c) "Neo4j"
 *  Neo4j Sweden AB [https://neo4j.com]
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package client

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"sync"
)

const (
	auraBasePath = "https://api.neo4j.io"
	auraV1Path   = auraBasePath + "/v1"
	userAgent    = "AuraTerraform/v0.0.1"
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
			mutex:        &sync.Mutex{},
		},
		httpClient: httpClient,
	}
}

func (c *AuraClient) Get(path string) ([]byte, int, error) {
	return c.doOperation("GET", path)
}

func (c *AuraClient) Post(path string, payload []byte) ([]byte, int, error) {
	return c.doOperationWithPayload("POST", path, payload)
}

func (c *AuraClient) Delete(path string) ([]byte, int, error) {
	return c.doOperation("DELETE", path)
}

func (c *AuraClient) Patch(path string, payload []byte) ([]byte, int, error) {
	return c.doOperationWithPayload("PATCH", path, payload)
}

func (c *AuraClient) doOperationWithPayload(method string, path string, payload []byte) ([]byte, int, error) {
	token, err := c.auth.GetToken()
	if err != nil {
		return []byte{}, 0, err
	}

	postUrl, err := url.Parse(auraV1Path + "/" + path)
	if err != nil {
		return []byte{}, 0, err
	}

	req := &http.Request{
		Method: method,
		URL:    postUrl,
		Header: map[string][]string{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer " + token},
			"User-Agent":    {userAgent},
		},
		Body: io.NopCloser(bytes.NewReader(payload)),
	}

	resp, err := c.httpClient.Do(req)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return []byte{}, 0, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, 0, err
	}
	return body, resp.StatusCode, nil
}

func (c *AuraClient) doOperation(method string, path string) ([]byte, int, error) {
	token, err := c.auth.GetToken()
	if err != nil {
		return []byte{}, 0, err
	}

	getUrl, err := url.Parse(auraV1Path + "/" + path)
	if err != nil {
		return []byte{}, 0, err
	}

	req := &http.Request{
		Method: method,
		URL:    getUrl,
		Header: map[string][]string{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer " + token},
			"User-Agent":    {userAgent},
		},
	}

	// todo retry
	resp, err := c.httpClient.Do(req)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return []byte{}, 0, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, 0, err
	}
	return body, resp.StatusCode, nil
}
