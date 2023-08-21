package apis

import (
	"github.com/robfig/cron"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

var (
	DefaultSyncJobSchedule     = "@hourly"
	DefaultSyncJobAuditLogPath = filepath.Join(os.TempDir(), "sync-audit.log")
)

// SyncJobSpec defines a store sync CR.
type SyncJobSpec struct {
	// Used to configure the source for sync request.
	// Required
	// TODO: This should reference a SecretStore CR.
	SourceStore SecretStoreSpec `json:"source-store"`

	// Used to configure the destination for sync request.
	// Required
	// TODO: This should reference a SecretStore CR.
	DestStore SecretStoreSpec `json:"dest-store"`

	// Used to configure keys to sync.
	// Optional
	Keys []StoreKey `json:"keys,omitempty"`

	// Used to configure regex filters to apply on listed keys.
	// Keys will not be filtered.
	// Defaults to empty.
	// Optional
	KeyFilters []string `json:"key-filters,omitempty"`

	// Template is applied to every key struct before sync.
	// Must return a valid JSON StoreKey object.
	// Optional
	Template string `json:"template"`

	// Used to configure schedule for synchronization.
	// The schedule is in Cron format, see https://en.wikipedia.org/wiki/Cron
	// Defaults to @hourly
	// Optional
	Schedule string `json:"schedule,omitempty"`

	// Used to only perform sync once.
	// If specified, Schedule will be ignored.
	// Optional
	RunOnce bool `json:"run-once,omitempty"`

	// Points to a file where sync logs should be saved to.
	// Defaults to DefaultSyncJobAuditLogPath
	// Optional
	// TODO: Implement support for audit log file.
	//  Only write successful key syncs to this file.
	//  Consider exposing String() to get basic API details on StoreClient.
	AuditLogPath string `json:"audit-log-path"`
}

func (spec *SyncJobSpec) GetSchedule() string {
	// Validate
	if spec.Schedule == "" {
		return DefaultSyncJobSchedule
	}
	if _, err := cron.Parse(spec.Schedule); err != nil {
		logrus.Errorf("using default Schedule %s due to parse error: %v", DefaultSyncJobSchedule, err)
		return DefaultSyncJobSchedule
	}

	return spec.Schedule
}

// GetTemplate returns a template to apply for a StoreKey on sync. Returns nil if not cofigured.
func (spec *SyncJobSpec) GetTemplate() *template.Template {
	if spec.Template == "" {
		return nil
	}

	tpl, err := template.New("template").Parse(spec.Template)
	if err != nil {
		logrus.Errorf("using nil Template due to parse error: %v", err)
		return nil
	}
	return tpl
}

func (spec *SyncJobSpec) GetAuditLogPath() string {
	if spec.AuditLogPath == "" {
		return DefaultSyncJobAuditLogPath
	}
	return spec.AuditLogPath
}

// StoreKey defines a common key-specific data for StoreClient.
type StoreKey struct {
	// Key points to a specific key in store.
	// Format "path/to/key"
	// Required
	Key string `json:"key"`

	// Version points to specific key version.
	Version string `json:"version"`
}

// GetPath returns relative path, e.g. GetPath("path/to/key") returns ["path", "to"]
func (key *StoreKey) GetPath() []string {
	parts := strings.Split(key.Key, "/")
	if len(parts) == 0 {
		return nil
	}
	return parts[:len(parts)-1]
}

// GetProperty returns property key at, e.g. GetProperty("path/to/key") returns "key"
func (key *StoreKey) GetProperty() string {
	parts := strings.Split(key.Key, "/")
	if len(parts) == 0 {
		return key.Key
	}
	return parts[len(parts)-1]
}

// SyncItemStatus defines status data for SyncJobSpec.
type SyncItemStatus struct {
	RefreshStatus []SyncJobRefreshStatus `json:"refresh-status"`
	LastSyncedAt  time.Time              `json:"last-synced-at"`
}

// SyncJobRefreshStatus defines data for a single refresh cycle.
type SyncJobRefreshStatus struct {
	Success  bool      `json:"success"`
	Status   string    `json:"status"`
	SyncedAt time.Time `json:"synced-at"`
}
