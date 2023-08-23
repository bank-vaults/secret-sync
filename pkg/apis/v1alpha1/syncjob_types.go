package v1alpha1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/robfig/cron"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

var (
	DefaultSyncJobSchedule     = "@hourly"
	DefaultSyncJobAuditLogPath = filepath.Join(os.TempDir(), "sync-audit.log")
	DefaultSyncJobHistoryLimit = 32
)

// SyncJobSpec defines a dest-source sync request CR.
type SyncJobSpec struct {
	// Used to configure the source for sync request.
	// Required
	SourceStore SecretStoreRef `json:"source-store"`

	// Used to configure the destination for sync request.
	// Required
	DestStore SecretStoreRef `json:"dest-store"`

	// Used to specify keys to sync.
	// Optional
	Keys []StoreKey `json:"keys,omitempty"`

	// Used to configure regex filters to apply on listed keys.
	// Keys will not be filtered.
	// Defaults to empty.
	// Optional
	ListFilters []string `json:"list-filters,omitempty"`

	// Template is applied to every key struct before sync.
	// Must return a valid JSON StoreKey object.
	// Optional
	Template string `json:"template,omitempty"`

	// Used to configure schedule for synchronization.
	// The schedule is in Cron format, see https://en.wikipedia.org/wiki/Cron
	// Defaults to @hourly
	// Optional
	Schedule string `json:"schedule,omitempty"`

	// Used to only perform sync once.
	// If specified, Schedule will be ignored.
	// Optional
	RunOnce bool `json:"run-once,omitempty"`

	// The number of sync results to retain.
	// Defaults to 32.
	// Optional
	HistoryLimit *int32 `json:"history-limit,omitempty"`

	// Points to a file where all sync logs should be saved to.
	// Defaults to DefaultSyncJobAuditLogPath
	// Optional
	// TODO: Implement support for audit log file.
	//  Only write successful key syncs to this file.
	//  Consider exposing String() to get basic API details on v1alpha1.StoreClient.
	AuditLogPath string `json:"audit-log-path,omitempty"`
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

func (spec *SyncJobSpec) GetAuditLogPath() string {
	if spec.AuditLogPath == "" {
		return DefaultSyncJobAuditLogPath
	}
	return spec.AuditLogPath
}

func (spec *SyncJobSpec) GetHistoryLimit() int32 {
	if spec.HistoryLimit == nil || *spec.HistoryLimit <= 0 {
		return int32(DefaultSyncJobHistoryLimit)
	}
	return *spec.HistoryLimit
}

func (spec *SyncJobSpec) GetListFilters() []*regexp.Regexp {
	regexFilters := make([]*regexp.Regexp, 0, len(spec.ListFilters))
	for _, filter := range spec.ListFilters {
		regexFilter, err := regexp.Compile(filter)
		if err != nil {
			logrus.Errorf("skipped filter %s due to parse error: %v", filter, err)
			continue
		}
		regexFilters = append(regexFilters, regexFilter)
	}
	return regexFilters
}

// ConvertKey converts a key based on the specified Template. Returns the same key if
// Template is not configured.
func (spec *SyncJobSpec) ConvertKey(key StoreKey) (*StoreKey, error) {
	// Get template
	if spec.Template == "" {
		return &key, nil
	}
	tpl, err := template.New("template").Parse(spec.Template)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	// Apply template
	buffer := &bytes.Buffer{}
	if err := tpl.Execute(buffer, key); err != nil {
		return nil, fmt.Errorf("failed to run template: %w", err)
	}

	// Parse key from template response
	var updatedKey StoreKey
	if err := json.Unmarshal(buffer.Bytes(), &updatedKey); err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}
	return &updatedKey, nil
}

// StoreKey defines common key-specific data.
type StoreKey struct {
	// Key points to a specific key in store.
	// Format "path/to/key"
	// Required
	Key string `json:"key"`

	// Version points to specific key version.
	// Optional
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

// SyncJobStatus defines status data for sync request CRs.
type SyncJobStatus struct {
	// Used to describe latest sync results.
	Responses []SyncJobStatusResponse `json:"conditions"`

	// Used to describe last successful sync request.
	LastSyncedAt string `json:"last-synced-at"`
}

// SyncJobStatusResponse defines data for a single sync result.
type SyncJobStatusResponse struct {
	Success  bool   `json:"success"`
	Status   string `json:"status"`
	SyncedAt string `json:"synced-at"`
}
