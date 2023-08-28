package v1alpha1

import "strings"

// SecretKey defines Provider key params.
// TODO: Add support for different encodings (to decode when fetching).
type SecretKey struct {
	// Key points to a specific key in store.
	// Format "path/to/key"
	// Required
	Key string `json:"key"`

	// Version points to specific key version.
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

// GetProperty returns property (domain) pointed by Key, e.g. GetProperty("path/to/key") returns "key"
func (key *SecretKey) GetProperty() string {
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

	// Used to rewrite secret keys after getting them from the Provider.
	// Multiple Rewrite operations will be applied in FIFO order.
	// Optional
	Rewrite []SecretKeyRewrite `json:"rewrite,omitempty"`
}

// SecretKeyQuery defines how to query SecretKey.
type SecretKeyQuery struct {
	// A root path to start the find operations.
	// Optional
	Path *string `json:"path,omitempty"`

	// Finds secret based on the regex key.
	// Optional
	Key *RegexpQuery `json:"key,omitempty"`
}

// SecretKeyRewrite defines how to rewrite SecretKey.
type SecretKeyRewrite struct {
	// Used to rewrite SecretKey with regular expressions.
	// The resulting SecretKey will be the output of a regexp.ReplaceAll operation.
	Regexp *RegexpRewrite `json:"regexp,omitempty"`
}

type RegexpQuery struct {
	Regexp string `json:"regexp,omitempty"`
}

type RegexpRewrite struct {
	// TODO: Add a way to specify reference field (e.g. Version, Regexp, ...)
	// TODO: For now, only Key is updated

	// Used to define the regular expression of a re.Compiler.
	Source string `json:"source"`

	// Used to define the target pattern of a ReplaceAll operation.
	Target string `json:"target"`
}
