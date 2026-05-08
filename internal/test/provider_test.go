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
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/provider"
)

// testMockServer is the package-level mock server instance started by TestMain.
// Individual tests can call testMockServer.Reset(), testMockServer.SeedInstance(),
// and testMockServer.SeedSnapshot() to prepare state before each test step.
var testMockServer *MockServer

func TestMain(m *testing.M) {
	testMockServer = NewMockServer()
	os.Setenv("AURA_BASE_URL", testMockServer.URL())

	code := m.Run()

	testMockServer.Close()
	os.Unsetenv("AURA_BASE_URL")

	os.Exit(code)
}

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"neo4jaura": providerserver.NewProtocol6WithError(provider.New("test")()),
}

const defaultProviderConfig = `
provider "neo4jaura" {
  client_id     = "test-client-id"
  client_secret = "test-client-secret"
}
`
