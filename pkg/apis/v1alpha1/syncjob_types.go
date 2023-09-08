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

	// Used to specify the source strategy for data fetch.
	// Required
	DataFrom []StrategyDataFrom `json:"dataFrom,omitempty"`

	// Used to specify the target strategy for data sync.
	// Required
	DataTo []StrategyDataTo `json:"dataTo,omitempty"`
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

// StrategyDataFrom defines how to fetch SecretRef from source store.
// TODO: Add support for specifying different source.
type StrategyDataFrom struct {
	// Used to define unique name for templating.
	// Required
	Name string `json:"name"`

	// Used to find secret key based on reference.
	// Optional, but SecretQuery must be provided
	SecretRef *SecretRef `json:"secretRef,omitempty"`

	// Used to find secret key based on query.
	// Optional, but SecretRef must be provided
	SecretQuery *SecretRefQuery `json:"secretQuery,omitempty"`
}

// StrategyDataTo defines how to sync fetched SecretRef to target store.
type StrategyDataTo struct {
	// Used to define target key. Supports templating.
	// Required
	Key string `json:"key"`

	// Used to define the resulting value. Supports templating.
	// Optional, but ValueMap must be provided
	Value *string `json:"value,omitempty"`

	// Used to define the resulting value map. Supports templating.
	// Optional, but Value must be provided
	ValueMap map[string]string `json:"valueMap,omitempty"`
}
