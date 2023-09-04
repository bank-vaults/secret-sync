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

package file

import (
	"context"
	"fmt"
	"github.com/bank-vaults/secret-sync/pkg/apis/v1alpha1"
)

type Provider struct{}

func (p *Provider) NewClient(_ context.Context, backend v1alpha1.SecretStoreProvider) (v1alpha1.StoreClient, error) {
	return &client{
		dir: backend.File.DirPath,
	}, nil
}

func (p *Provider) Validate(backend v1alpha1.SecretStoreProvider) error {
	if backend.File == nil {
		return fmt.Errorf("empty .File")
	}
	if backend.File.DirPath == "" {
		return fmt.Errorf("empty .File.DirPath")
	}
	return nil
}

func init() {
	v1alpha1.Register(&Provider{}, &v1alpha1.SecretStoreProvider{
		File: &v1alpha1.SecretStoreProviderFile{},
	})
}
