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

package test

import (
	"context"
	"os"

	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/client"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func executeCypher(ctx context.Context, connectionUrl, username, password string, query string) error {
	driver, err := neo4j.NewDriverWithContext(connectionUrl, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return err
	}
	defer driver.Close(ctx)

	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err = session.ExecuteWrite(ctx, func(transaction neo4j.ManagedTransaction) (any, error) {
		result, err := transaction.Run(ctx, query, nil)
		if err != nil {
			return nil, err
		}
		return result.Consume(ctx)
	})
	return err
}

func newTestAuraApi() *client.AuraApi {
	return client.NewAuraApi(
		client.NewAuraClient(
			os.Getenv("TF_VAR_client_id"),
			os.Getenv("TF_VAR_client_secret"),
			"0.0.0-tests"),
		nil, nil)
}
