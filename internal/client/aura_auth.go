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
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

const expirationBuffer = 60

type AuraAuth struct {
	clientId     string
	clientSecret string
	mutex        *sync.Mutex
	token        *AuraAuthToken
	httpClient   *retryablehttp.Client
	userAgent    string
}

type AuraAuthToken struct {
	token      string
	expiringAt int64
}

func (a *AuraAuth) authenticate(ctx context.Context) error {
	authUrl := fmt.Sprintf("%s/%s", auraBasePath, "oauth/token")
	req, err := retryablehttp.NewRequestWithContext(ctx, "POST", authUrl, []byte("grant_type=client_credentials"))
	if err != nil {
		return err
	}
	req.SetBasicAuth(a.clientId, a.clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", a.userAgent)

	resp, err := a.httpClient.Do(req)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("authentication failed with no response")
	}

	//Check response status
	if resp.StatusCode != 200 {
		return fmt.Errorf("authentication failed with status %d.  Check client id and secret values",
			resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var token TokenResponse
	err = json.Unmarshal(body, &token)
	if err != nil {
		return err
	}

	a.token = &AuraAuthToken{
		token:      token.AccessToken,
		expiringAt: time.Now().Unix() + token.ExpiresIn,
	}

	return nil
}

func (a *AuraAuth) GetToken(ctx context.Context) (string, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if a.token == nil || a.token.expiringAt <= time.Now().Unix()+expirationBuffer {
		err := a.authenticate(ctx)
		if err != nil {
			return "", err
		}
	}
	return a.token.token, nil
}
