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

package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/ghodss/yaml"
	"github.com/krayzpipes/cronticker/cronticker"
	"github.com/spf13/cobra"

	"github.com/bank-vaults/secret-sync/pkg/apis/v1alpha1"
	"github.com/bank-vaults/secret-sync/pkg/provider"
	"github.com/bank-vaults/secret-sync/pkg/storesync"
)

const (
	flagSource   = "source"
	flagTarget   = "target"
	flagSyncJob  = "syncjob"
	flagSchedule = "schedule"
)

type syncJob struct {
	source *v1alpha1.StoreClient
	target *v1alpha1.StoreClient
	sync   *v1alpha1.SyncPlan
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronizes secrets from a source to a target store based on sync strategy.",
	RunE:  run,
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.PersistentFlags().StringP(flagSource, "s", "", "Source store config file. ")
	_ = syncCmd.MarkPersistentFlagRequired(flagSource)
	syncCmd.PersistentFlags().StringP(flagTarget, "t", "", "Target store config file. ")
	_ = syncCmd.MarkPersistentFlagRequired(flagTarget)
	syncCmd.PersistentFlags().String(flagSyncJob, "", "Sync job config file. ")
	_ = syncCmd.MarkPersistentFlagRequired(flagSyncJob)

	syncCmd.PersistentFlags().String(flagSchedule, "", "Sync periodically using CRON schedule. If not specified, runs only once.")
}

func run(cmd *cobra.Command, args []string) error {
	syncJob, err := prepareSync(cmd, args)
	if err != nil {
		return fmt.Errorf("failed to prepare sync job: %w", err)
	}

	// Run once
	if syncJob.sync.GetSchedule(cmd.Root().Context()) == nil {
		resp, err := storesync.Sync(cmd.Root().Context(), *syncJob.source, *syncJob.target, syncJob.sync.Sync)
		if err != nil {
			return fmt.Errorf("failed to sync secrets: %w", err)
		}
		slog.InfoContext(cmd.Root().Context(), resp.Status)
		return nil
	}

	// Run on schedule
	cronTicker, err := cronticker.NewTicker(syncJob.sync.Schedule)
	if err != nil {
		return fmt.Errorf("failed to create CRON ticker: %w", err)
	}
	defer cronTicker.Stop()

	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, os.Interrupt)
	for {
		select {
		case <-cronTicker.C:
			slog.InfoContext(cmd.Root().Context(), "Handling a new sync request...")

			resp, err := storesync.Sync(cmd.Root().Context(), *syncJob.source, *syncJob.target, syncJob.sync.Sync)
			if err != nil {
				return err
			}
			slog.InfoContext(cmd.Root().Context(), resp.Status)

		case <-cancel:
			return nil
		}
	}
}

func prepareSync(cmd *cobra.Command, _ []string) (*syncJob, error) {
	// Init source
	source, err := cmd.Flags().GetString(flagSource)
	if err != nil {
		return nil, fmt.Errorf("failed to get source flag: %w", err)
	}

	sourceStore, err := loadStore(source)
	if err != nil {
		return nil, fmt.Errorf("failed to load source store: %w", err)
	}

	sourceProvider, err := provider.NewClient(cmd.Root().Context(), sourceStore)
	if err != nil {
		return nil, fmt.Errorf("failed to create source client: %w", err)
	}

	// Init target
	target, err := cmd.Flags().GetString(flagTarget)
	if err != nil {
		return nil, fmt.Errorf("failed to get target flag: %w", err)
	}

	targetStore, err := loadStore(target)
	if err != nil {
		return nil, fmt.Errorf("failed to load target store: %w", err)
	}

	targetProvider, err := provider.NewClient(cmd.Root().Context(), targetStore)
	if err != nil {
		return nil, fmt.Errorf("failed to create target client: %w", err)
	}

	// Init sync request by loading from file and overriding from cli
	sync, err := cmd.Flags().GetString(flagSyncJob)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync flag: %w", err)
	}

	syncPlan, err := loadSyncPlan(sync)
	if err != nil {
		return nil, fmt.Errorf("failed to load sync plan: %w", err)
	}

	schedule, _ := cmd.Flags().GetString(flagSchedule)
	if schedule != "" {
		syncPlan.Schedule = schedule
	}

	return &syncJob{
		source: &sourceProvider,
		target: &targetProvider,
		sync:   syncPlan,
	}, nil
}

func loadStore(path string) (*v1alpha1.SecretStoreSpec, error) {
	// Load file
	yamlBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Unmarshal (convert YAML to JSON)
	var storeConfig = struct {
		SecretsStore v1alpha1.SecretStoreSpec `json:"secretsStore"`
	}{}

	jsonBytes, err := yaml.YAMLToJSON(yamlBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to convert YAML to JSON: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, &storeConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return &storeConfig.SecretsStore, nil
}

func loadSyncPlan(path string) (*v1alpha1.SyncPlan, error) {
	// Load file
	yamlBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Unmarshal (convert YAML to JSON)
	var ruleCfg v1alpha1.SyncPlan

	jsonBytes, err := yaml.YAMLToJSON(yamlBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to convert YAML to JSON: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, &ruleCfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return &ruleCfg, nil
}
