// Copyright © 2023 Cisco
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	"os"
	"path/filepath"

	"github.com/robfig/cron"
	"github.com/sirupsen/logrus"
)

var (
	DefaultSyncJobSchedule     = "@hourly"
	DefaultSyncJobAuditLogPath = filepath.Join(os.TempDir(), "sync-audit.log")
)

// SyncJob defines a source-to-dest sync request.
// TODO: Add support for auditing.
type SyncJob struct {
	// Points to a file where all sync logs should be saved to.
	// Defaults to DefaultSyncJobAuditLogPath
	// Optional
	AuditLogPath string `json:"auditLogPath,omitempty"`

	// Used to configure schedule for synchronization.
	// The schedule is in Cron format, see https://en.wikipedia.org/wiki/Cron
	// Defaults to @hourly
	// Optional
	Schedule string `json:"schedule,omitempty"`

	// Used to only perform sync once.
	// If specified, Schedule will be ignored.
	// Optional
	RunOnce bool `json:"runOnce,omitempty"`

	// Used to specify the strategy for secrets sync.
	// Required
	Sync []SyncItem `json:"sync,omitempty"`
}

func (spec *SyncJob) GetSchedule() string {
	if spec.Schedule == "" {
		return DefaultSyncJobSchedule
	}
	if _, err := cron.Parse(spec.Schedule); err != nil {
		logrus.Errorf("using default Schedule %s due to parse error: %v", DefaultSyncJobSchedule, err)
		return DefaultSyncJobSchedule
	}

	return spec.Schedule
}

func (spec *SyncJob) GetAuditLogPath() string {
	if spec.AuditLogPath == "" {
		return DefaultSyncJobAuditLogPath
	}
	return spec.AuditLogPath
}

// SecretsSelector defines a secret selector for a given ref or query.
// This enables named usage in templates given as:
// a) when using FromRef, enables {{ .Data.ref_name }}
// b) when using FromQuery, enables {{ .Data.query_name.<SECRET_KEY> }}
type SecretsSelector struct {
	// Used to define unique name for templating.
	// Required
	Name string `json:"name"`

	// FromRef selects a secret from a reference.
	// Optional, but SecretQuery must be provided
	FromRef *SecretRef `json:"fromRef,omitempty"`

	// FromQuery selects secret(s) from a query.
	// Optional, but SecretRef must be provided
	FromQuery *SecretQuery `json:"fromQuery,omitempty"`
}

// SyncTarget defines where the secret(s) will be synced to.
type SyncTarget struct {
	// Key indicates that a single SecretRef will be synced to target.
	Key *string `json:"key,omitempty"`

	// KeyPrefix indicates that multiple SecretRef will be synced to target.
	KeyPrefix *string `json:"keyPrefix,omitempty"`
}

// SyncTemplate defines how to obtain SecretRef using template.
type SyncTemplate struct {
	// Used to define the resulting secret (raw) value. Supports templating.
	// Optional, but Data must be provided
	RawData *string `json:"rawData,omitempty"`

	// Used to define the resulting secret (map) value. Supports templating.
	// Optional, but RawData must be provided
	Data map[string]string `json:"data,omitempty"`
}

// SyncItem defines how to fetch from source, transform, and sync SecretRef(s) on target.
type SyncItem struct {
	// FromRef selects a secret from a reference.
	// SyncTarget.Key must be specified.
	FromRef *SecretRef `json:"fromRef,omitempty"`

	// FromQuery selects secret(s) from a query.
	// To sync one secret, SyncTarget.Key and Template must be specified.
	// To sync all secrets from query, SyncTarget.KeyPrefix must be specified.
	FromQuery *SecretQuery `json:"fromQuery,omitempty"`

	// FromSources select secret(s) from a multiple sources.
	// SyncTarget.Key must be specified.
	FromSources []SecretsSelector `json:"fromSources,omitempty"`

	// Target defines where the key(s) from sources will be synced on target.
	Target SyncTarget `json:"target"`

	// Template defines how the fetched key(s) will be transformed to create a new
	// SecretRef that will be synced to target.
	Template *SyncTemplate `json:"template,omitempty"`
}
