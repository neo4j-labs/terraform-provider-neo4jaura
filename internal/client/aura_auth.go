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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const expirationBuffer = 60

type AuraAuth struct {
	clientId     string
	clientSecret string
	mutex        *sync.Mutex
	token        *AuraAuthToken
	httpClient   *http.Client
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

	encodedCreds := base64.StdEncoding.EncodeToString([]byte(a.clientId + ":" + a.clientSecret))

	req := &http.Request{
		Method: "POST",
		URL:    authUrl,
		Header: map[string][]string{
			"Content-Type":  {"application/x-www-form-urlencoded"},
			"Authorization": {"Basic " + encodedCreds},
		},
		Body: io.NopCloser(strings.NewReader("grant_type=client_credentials")),
	}

	// todo retry
	resp, err := a.httpClient.Do(req)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
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

func (a *AuraAuth) GetToken() (string, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if a.token == nil || a.token.expiringAt <= time.Now().Unix()+expirationBuffer {
		err := a.authenticate()
		if err != nil {
			return "", err
		}
	}
	return a.token.token, nil
}
