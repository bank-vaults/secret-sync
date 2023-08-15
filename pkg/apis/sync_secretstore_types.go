package apis

import (
	"strings"
	"text/template"
	"time"
)

var DefaultSyncRequestPeriod = time.Hour

// SyncSecretStoreSpec defines a store sync CR.
type SyncSecretStoreSpec struct {
	// Used to configure the source for sync request.
	// Required
	// TODO: This should reference a SecretStore CR.
	SourceStore *SecretStoreSpec `json:"source-store"`

	// Used to configure the destination for sync request.
	// Required
	// TODO: This should reference a SecretStore CR.
	DestStore *SecretStoreSpec `json:"dest-store"`

	// Used to configure keys to sync.
	// Optional
	Keys []StoreKey `json:"keys,omitempty"`

	// Used to configure regex filters to apply on listed keys.
	// Keys will not be filtered.
	// Defaults to empty.
	// Optional
	KeyFilters []string `json:"key-filters,omitempty"`

	// SyncTemplate is applied to every key struct before sync.
	// Must return a valid JSON StoreKey object.
	// Optional
	SyncTemplate string `json:"template"`

	// Used to configure period for synchronization.
	// Defaults to 1h.
	// Optional
	SyncPeriod string `json:"period,omitempty"`

	// Used to only perform sync once.
	// If specified, SyncPeriod will be ignored.
	// Optional
	SyncOnce bool `json:"sync-once,omitempty"`
}

func (spec *SyncSecretStoreSpec) GetSyncPeriod() time.Duration {
	if spec.SyncPeriod == "" {
		return DefaultSyncRequestPeriod
	}

	duration, err := time.ParseDuration(spec.SyncPeriod)
	if err != nil {
		// log.Error(err, "using default SyncPeriod due to parse error", spec.SyncPeriod)
		return DefaultSyncRequestPeriod
	}
	return duration
}

func (spec *SyncSecretStoreSpec) GetSyncTemplate() *template.Template {
	if spec.SyncTemplate == "" {
		return nil
	}
	
	tpl, err := template.New("template").Parse(spec.SyncTemplate)
	if err != nil {
		// log.Error(err, "using nil Template due to parse error", spec.SyncTemplate)
		return nil
	}
	return tpl
}

// StoreKey defines a common key-specific data for StoreClient.
type StoreKey struct {
	// Key points to a specific key in store.
	// Standard: "path/to/key"
	// Required
	Key string `json:"key"`

	// Version points to specific key version.
	Version string `json:"version"`
}

// GetPath returns path to Key, e.g. GetPath("path/to/key") returns ["path", "to"]
func (key *StoreKey) GetPath() []string {
	parts := strings.Split(key.Key, "/")
	if len(parts) == 0 {
		return nil
	}
	return parts[:len(parts)-1]
}

// GetProperty returns property key points at, e.g. GetProperty("path/to/key") returns "key"
func (key *StoreKey) GetProperty() string {
	parts := strings.Split(key.Key, "/")
	if len(parts) == 0 {
		return key.Key
	}
	return parts[len(parts)-1]
}
