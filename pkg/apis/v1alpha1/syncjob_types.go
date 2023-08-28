package v1alpha1

import (
	"github.com/robfig/cron"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

var (
	DefaultSyncJobSchedule     = "@hourly"
	DefaultSyncJobAuditLogPath = filepath.Join(os.TempDir(), "sync-audit.log")
	DefaultSyncJobHistoryLimit = 32
)

// SyncJobSpec defines a source-to-dest sync request CR.
type SyncJobSpec struct {
	// Used to configure the source for sync request.
	// Required
	SourceRef SecretStoreRef `json:"source"`

	// Used to configure the destination for sync request.
	// Required
	DestRef SecretStoreRef `json:"dest"`

	// Used to configure schedule for synchronization.
	// The schedule is in Cron format, see https://en.wikipedia.org/wiki/Cron
	// Defaults to @hourly
	// Optional
	Schedule string `json:"schedule,omitempty"`

	// Used to only perform sync once.
	// If specified, Schedule will be ignored.
	// Optional
	RunOnce bool `json:"run-once,omitempty"`

	// Used to specify sync plan.
	// Required
	Plan []SecretKeyFromRef `json:"plan,omitempty"`

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
