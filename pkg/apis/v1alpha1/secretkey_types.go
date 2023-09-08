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

// SecretRef defines Provider reference key.
// TODO: Add support for version
// TODO: Add support for map field selector
// TODO: Add support for encoding
type SecretRef struct {
	// Key points to a specific key in store.
	// Format "path/to/key"
	// Required
	Key string `json:"key"`

	// Version points to specific key version.
	// Optional
	Version *string `json:"version,omitempty"`
}

// GetPath returns path pointed by Key, e.g. GetPath("/path/to/key") returns ["path", "to"]
func (key *SecretRef) GetPath() []string {
	parts := strings.Split(key.Key, "/")
	if len(parts) == 0 {
		return nil
	}
	return parts[:len(parts)-1]
}

// GetProperty returns property (domain) pointed by Key, e.g. GetProperty("/path/to/key") returns "key"
func (key *SecretRef) GetProperty() string {
	parts := strings.Split(key.Key, "/")
	if len(parts) == 0 {
		return key.Key
	}
	return parts[len(parts)-1]
}

// SecretRefQuery defines how to query Provider to obtain SecretRef.
type SecretRefQuery struct {
	// A root path to start the query operations.
	// Optional
	Path *string `json:"path,omitempty"`

	// Finds SecretRef based on the regexp params.
	// Required
	Regexp QueryRegexp `json:"regexp"`
}

// QueryRegexp defines how to search for a SecretRef using regexp query.
// TODO: Add support for version
// TODO: Add support for map field selector
type QueryRegexp struct {
	// Searches SecretRef based on key query
	// Required
	Key string `json:"key"`
}
