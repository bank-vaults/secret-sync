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

import (
	"fmt"
	"strings"
)

var ErrKeyPathUnsupported = fmt.Errorf("provider does not support secret keys with path")

// SecretKey defines Provider key params.
// TODO: Add support for different encodings (to decode when fetching).
type SecretKey struct {
	// Key points to a specific key in store.
	// Accepted formats: "key", "path/to/key".
	// Some Provider do not support "path/to/key" and will throw ErrKeyPathUnsupported.
	// Required
	Key string `json:"key"`

	// Property selects a specific map property of the Provider value.
	// TODO: Add support on providers
	// Optional
	Property string `json:"property,omitempty"`

	// Version points to specific key version.
	// TODO: Add support on providers
	// Optional
	Version string `json:"version"`
}

// GetPath returns path pointed by Key, e.g. GetPath("path/to/key") returns ["path", "to"]
func (key *SecretKey) GetPath() []string {
	parts := strings.Split(key.Key, "/")
	if len(parts) == 0 {
		return nil
	}
	return parts[:len(parts)-1]
}

// GetKey returns base key pointed by Key, e.g. GetKey("path/to/key") returns "key"
func (key *SecretKey) GetKey() string {
	parts := strings.Split(key.Key, "/")
	if len(parts) == 0 {
		return key.Key
	}
	return parts[len(parts)-1]
}

// SecretKeyFromRef defines SecretKey data to fetch and transform from referenced store.
// TODO: Add support for overriding default SyncJob source.
type SecretKeyFromRef struct {
	// Used to reference a static secret key.
	// Optional
	SecretKey *SecretKey `json:"secret,omitempty"`

	// Used to find secret key based on query.
	// Ignored if SecretKey is specified.
	// Optional
	Query *SecretKeyQuery `json:"query,omitempty"`

	// Used to transform secret keys after getting them from the Provider.
	// Multiple KeyTransform operations will be applied in FIFO order.
	// Optional
	KeyTransform []SecretKeyTransform `json:"key-transform,omitempty"`
}

type SecretKeyQuery struct {
	// A root path to start the find operations.
	// Optional
	Path *string `json:"path,omitempty"`

	// Finds secret based on the regex key.
	// Optional
	Key *RegexpQuery `json:"key,omitempty"`
}

type SecretKeyTransform struct {
	// Used to transform SecretKey with regular expressions.
	// The resulting SecretKey will be the output of a regexp.ReplaceAll operation.
	Regexp *RegexpTransform `json:"regexp,omitempty"`
}

type RegexpQuery struct {
	Regexp string `json:"regexp,omitempty"`
}

type RegexpTransform struct {
	// Used to define the regular expression of a re.Compiler.
	Source string `json:"source"`

	// Used to define the target pattern of a ReplaceAll operation.
	Target string `json:"target"`
}
