package v1alpha1

// SecretStoreProviderVault defines provider for a Vault.
type SecretStoreProviderVault struct {
	Address        string `json:"address"`
	UnsealKeysPath string `json:"unseal-keys-path"`
	Role           string `json:"role"`
	AuthPath       string `json:"auth-path"`
	TokenPath      string `json:"token-path"`
	Token          string `json:"token"` // TODO: Add support for reading this from a k8s secret
}
