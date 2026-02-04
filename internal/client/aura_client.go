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
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

const (
	auraBasePath = "https://api.neo4j.io"
	auraV1Path   = auraBasePath + "/v1"
	userAgent    = "AuraTerraform/v0.0.3"
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

func (c *AuraClient) Get(ctx context.Context, path string) ([]byte, int, error) {
	return c.doOperation(ctx, "GET", path, nil)
}

func (c *AuraClient) Post(ctx context.Context, path string, payload []byte) ([]byte, int, error) {
	return c.doOperation(ctx, "POST", path, payload)
}

func (c *AuraClient) Delete(ctx context.Context, path string) ([]byte, int, error) {
	return c.doOperation(ctx, "DELETE", path, nil)
}

func (c *AuraClient) Patch(ctx context.Context, path string, payload []byte) ([]byte, int, error) {
	return c.doOperation(ctx, "PATCH", path, payload)
}

func (c *AuraClient) doOperation(ctx context.Context, method string, path string, payload []byte) ([]byte, int, error) {
	token, err := c.auth.GetToken(ctx)
	if err != nil {
		return []byte{}, 0, err
	}

	absoluteUrl := fmt.Sprintf("%s/%s", auraV1Path, path)

	req, err := retryablehttp.NewRequestWithContext(ctx, method, absoluteUrl, payload)
	if err != nil {
		return []byte{}, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("User-Agent", userAgent)

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
