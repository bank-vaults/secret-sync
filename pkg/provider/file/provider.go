package file

import (
	"context"
	"fmt"
	"github.com/bank-vaults/secret-sync/pkg/apis"
)

type Provider struct{}

var _ apis.Provider = &Provider{}

func (p *Provider) NewClient(_ context.Context, store apis.SecretStoreSpec) (apis.StoreClient, error) {
	provider := store.Provider.File
	return &client{
		dir: provider.ParentDir,
	}, nil
}

func (p *Provider) Validate(store apis.SecretStoreSpec) error {
	provider := store.Provider.File
	if provider == nil {
		return fmt.Errorf("empty .File")
	}
	if provider.ParentDir == "" {
		return fmt.Errorf("empty .File.ParentDir")
	}
	return nil
}

func init() {
	apis.Register(&Provider{}, &apis.SecretStoreProvider{
		File: &apis.SecretStoreProviderFile{},
	})
}
