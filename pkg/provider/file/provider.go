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
