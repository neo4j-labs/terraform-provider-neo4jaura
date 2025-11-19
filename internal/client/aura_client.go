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
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

const (
	auraBasePath = "https://api.neo4j.io"
	auraV1Path   = auraBasePath + "/v1"
	userAgent    = "AuraTerraform/v0.0.1"
)

const (
	maxRetries = 5
	backoffMin = 1 * time.Second
	backoffMax = 30 * time.Second
)

type AuraClient struct {
	auth       *AuraAuth
	httpClient *retryablehttp.Client
}

func NewAuraClient(clientId, clientSecret string) *AuraClient {
	httpClient := retryablehttp.NewClient()
	httpClient.RetryMax = maxRetries
	httpClient.RetryWaitMin = backoffMin
	httpClient.RetryWaitMax = backoffMax
	httpClient.CheckRetry = retryablehttp.DefaultRetryPolicy
	httpClient.Backoff = retryablehttp.DefaultBackoff

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

	req := &retryablehttp.Request{
		Request: &http.Request{
			Method: method,
			URL:    postUrl,
			Header: map[string][]string{
				"Content-Type":  {"application/json"},
				"Authorization": {"Bearer " + token},
				"User-Agent":    {userAgent},
			},
		},
	}
	if payload != nil {
		err = req.SetBody(payload)
		if err != nil {
			return []byte{}, 0, err
		}
	}

	resp, err := c.httpClient.Do(req)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return []byte{}, 0, err
	}
	if resp == nil {
		return []byte{}, 0, fmt.Errorf("no response from server")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, 0, err
	}
	return body, resp.StatusCode, nil
}

func (c *AuraClient) doOperation(method string, path string) ([]byte, int, error) {
	return c.doOperationWithPayload(method, path, nil)
}
