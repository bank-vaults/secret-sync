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

// SyncJobSpec defines a source-to-dest sync request CR.
type SyncJobSpec struct {
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
