package v1alpha1

import "strings"

var DefaultSecretStorePermissions = SecretStorePermissionsReadWrite

// SecretStoreRef defines a reference to a SecretStore.
type SecretStoreRef struct {
	// Name points to a name of a resource.
	Name string `json:"name,omitempty"`

	// Namespace points to a namespace of a resource.
	Namespace string `json:"namespace,omitempty"`
}

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

// SecretStoreProvider defines which store provider to use.
// Only one can be specified.
type SecretStoreProvider struct {
	// Used for Vault provider.
	Vault *SecretStoreProviderVault `json:"vault,omitempty"`

	// Used for non-encrypted File provider.
	File *SecretStoreProviderFile `json:"file,omitempty"`
}
