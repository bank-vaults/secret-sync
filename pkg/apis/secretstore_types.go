package apis

import (
	"strings"
)

var DefaultSecretStorePermissions = SecretStoreReadWrite

// SecretStoreSpec defines an arbitrary SecretStore CR.
type SecretStoreSpec struct {
	// Used to configure secrets provider.
	// Required
	Provider *SecretStoreProvider `json:"provider"`

	// Used to configure store mode. Defaults to ReadWrite.
	// Optional
	Permissions SecretStorePermissions `json:"permissions,omitempty"`
}

func (spec *SecretStoreSpec) GetPermissions() SecretStorePermissions {
	if spec.Permissions == "" {
		return DefaultSecretStorePermissions
	}
	return spec.Permissions
}

type SecretStoreProvider struct {
	// Used for Vault provider.
	Vault *SecretStoreProviderVault `json:"vault,omitempty"`

	// Used for non-encrypted File provider.
	File *SecretStoreProviderFile `json:"file,omitempty"`
}

type SecretStorePermissions string

const (
	SecretStoreRead      SecretStorePermissions = "Read"
	SecretStoreWrite     SecretStorePermissions = "Write"
	SecretStoreReadWrite SecretStorePermissions = "ReadWrite"
)

func (p SecretStorePermissions) CanPerform(perm SecretStorePermissions) bool {
	return strings.Contains(string(p), string(perm))
}
