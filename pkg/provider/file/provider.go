package file

import (
	"context"
	"fmt"
	"github.com/bank-vaults/secret-sync/pkg/apis"
)

type Provider struct{}

var _ apis.Provider = &Provider{}

func (p *Provider) NewClient(_ context.Context, store apis.SecretStoreSpec) (apis.StoreClient, error) {
	return &client{
		dir: store.Provider.File.ParentDir,
	}, nil
}

func (p *Provider) Validate(store apis.SecretStoreSpec) error {
	providerFile := store.Provider.File
	if providerFile == nil {
		return fmt.Errorf("empty .File")
	}
	if providerFile.ParentDir == "" {
		return fmt.Errorf("empty .File.ParentDir")
	}
	return nil
}

func init() {
	apis.Register(&Provider{}, &apis.SecretStoreProvider{
		File: &apis.SecretStoreProviderFile{},
	})
}
