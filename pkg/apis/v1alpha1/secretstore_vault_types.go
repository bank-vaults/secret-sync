// Copyright © 2023 Cisco
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

// SecretStoreProviderVault defines provider for a Vault.
type SecretStoreProviderVault struct {
	Address        string `json:"address"`
	UnsealKeysPath string `json:"unseal-keys-path"`
	Role           string `json:"role"`
	AuthPath       string `json:"auth-path"`
	TokenPath      string `json:"token-path"`
	Token          string `json:"token"` // TODO: Add support for reading this from a k8s secret
}
