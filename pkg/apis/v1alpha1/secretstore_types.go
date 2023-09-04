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

import "strings"

var DefaultSecretStorePermissions = SecretStorePermissionsReadWrite

// SecretStoreSpec defines an arbitrary SecretStore spec.
type SecretStoreSpec struct {
	// Used to configure store mode. Defaults to ReadWrite.
	// Optional
	Permissions SecretStorePermissions `json:"permissions,omitempty"`

	// Used to configure secrets provider.
	// Required
	Provider SecretStoreProvider `json:"provider"`
}

func (spec *SecretStoreSpec) GetPermissions() SecretStorePermissions {
	if spec.Permissions == "" {
		return DefaultSecretStorePermissions
	}
	return spec.Permissions
}

type SecretStorePermissions string

const (
	SecretStorePermissionsRead      SecretStorePermissions = "Read"
	SecretStorePermissionsWrite     SecretStorePermissions = "Write"
	SecretStorePermissionsReadWrite SecretStorePermissions = "ReadWrite"
)

func (p SecretStorePermissions) CanPerform(perm SecretStorePermissions) bool {
	return strings.Contains(string(p), string(perm))
}

// SecretStoreProvider defines secret backend for Provider.
// Only one can be specified.
type SecretStoreProvider struct {
	// Used for Vault provider.
	Vault *SecretStoreProviderVault `json:"vault,omitempty"`

	// Used for Kubernetes provider.
	Kubernetes *SecretStoreProviderKubernetes `json:"kubernetes,omitempty"`

	// Used for non-encrypted File provider.
	File *SecretStoreProviderFile `json:"file,omitempty"`
}
