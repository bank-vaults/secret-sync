package apis

type SecretStoreProviderVault struct {
	Address        string `json:"address"`
	UnsealKeysPath string `json:"unseal-keys-path"`
	Role           string `json:"role"`
	AuthPath       string `json:"auth-path"`
	TokenPath      string `json:"token-path"`
	Token          string `json:"token"`
}
