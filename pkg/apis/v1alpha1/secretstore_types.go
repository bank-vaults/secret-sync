package v1alpha1

import "strings"

var DefaultSecretStorePermissions = SecretStorePermissionsReadWrite

type SecretStoreRef struct {
	// Name of the SecretStore resource
	Name string `json:"name"`

	// Namespace of the SecretStore resource.
	// Optional
	Namespace string `json:"namespace,omitempty"`

	// Kind of the SecretStore resource (SecretStore, ClusterSecretStore).
	// Optional
	Kind string `json:"kind,omitempty"`
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

// SecretStoreProvider defines secret backend for Provider.
// Only one can be specified.
type SecretStoreProvider struct {
	// Used for Vault provider.
	Vault *SecretStoreProviderVault `json:"vault,omitempty"`

	// Used for non-encrypted File provider.
	File *SecretStoreProviderFile `json:"file,omitempty"`
}
