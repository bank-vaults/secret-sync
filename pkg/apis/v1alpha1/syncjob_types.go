// Copyright © 2024 Bank-Vaults Maintainers
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
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/robfig/cron"
)

var DefaultSyncJobAuditLogPath = filepath.Join(os.TempDir(), "sync-audit.log")

// SyncJob defines overall source-to-target sync strategy.
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

	// Used to specify the strategy for secrets sync.
	// Required
	Sync []SyncAction `json:"sync,omitempty"`
}

func (spec *SyncJob) GetSchedule() *string {
	if spec.Schedule == "" {
		return nil
	}
	if _, err := cron.Parse(spec.Schedule); err != nil {
		slog.Error(fmt.Errorf("skipping Schedule due to parse error: %w", err).Error())
		return nil
	}

	return &spec.Schedule
}

func (spec *SyncJob) GetAuditLogPath() string {
	if spec.AuditLogPath == "" {
		return DefaultSyncJobAuditLogPath
	}
	return spec.AuditLogPath
}

// SyncAction defines how to fetch, transform, and sync SecretRef(s) from source to target.
// Only one of FromRef, FromQuery, FromSources can be specified.
type SyncAction struct {
	// FromRef selects a secret from a reference.
	// If SyncTarget.Key is nil, it will sync under referenced key.
	// If SyncTarget.Key is not-nil, it will sync under targeted key.
	FromRef *SecretRef `json:"secretRef,omitempty"`

	// FromQuery selects secret(s) from a query.
	// To sync one secret, SyncTarget.Key and Template must be specified.
	// To sync all secrets, SyncTarget.KeyPrefix must be specified.
	FromQuery *SecretQuery `json:"secretQuery,omitempty"`

	// FromSources select secret(s) from a multiple sources.
	// SyncTarget.Key and Template must be specified.
	FromSources []SecretSource `json:"secretSources,omitempty"`

	// Target defines where the key(s) from sources will be synced on target.
	// SyncTarget.Key means that only one secret will be synced.
	// SyncTarget.KeyPrefix means that multiple secrets will be synced.
	Target SyncTarget `json:"target,omitempty"`

	// Flatten indicates secrets FromQuery will be synced to a single SyncTarget.Key.
	Flatten *bool `json:"flatten,omitempty"`

	// Template defines how the fetched key(s) will be transformed to create a new
	// SecretRef that will be synced to target.
	// When using FromRef, {{ .Data }} defines given secrets raw value.
	// When using FromQuery and SyncTarget.Key, specific <KEY> raw values can be accessed via {{ .Data.<KEY> }}.
	// When using FromQuery and SyncTarget.KeyPrefix, {{ .Data }} defines raw values of query iterator.
	// When using FromSources, specific <NAMED SOURCE> secret data can be accessed via {{ .Data.<NAMED SOURCE> }}.
	Template *SyncTemplate `json:"template,omitempty"`
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
