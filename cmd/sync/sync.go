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

package sync

import (
	"context"
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

func NewSyncCmd(ctx context.Context) *cobra.Command {
	// Create cmd
	cmd := &syncCmd{}
	cobraCmd := &cobra.Command{
		Use:   "sync",
		Short: "Synchronizes secrets from a source to a target store based on sync strategy.",
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := cmd.init(); err != nil {
				return fmt.Errorf("error initializing sync command: %w", err)
			}

			return cmd.run(cmd.sync)
		},
	}
	// ctx passed to project components
	cmd.ctx = ctx
	// ctx passed to cobra
	cobraCmd.SetContext(ctx)

	// Register cmd flags
	cobraCmd.Flags().StringVar(&cmd.flgSrcFile, "source", "", "Source store config file. "+
		"This is the store where the data will be fetched from.")
	_ = cobraCmd.MarkFlagRequired("source")
	cobraCmd.Flags().StringVar(&cmd.flagDstFile, "target", "", "Target store config file. "+
		"This is the store where the data will be synced to.")
	_ = cobraCmd.MarkFlagRequired("target")
	cobraCmd.Flags().StringVar(&cmd.flagSyncFile, "sync", "", "Sync job config file. "+
		"This is the strategy sync template.")
	_ = cobraCmd.MarkFlagRequired("sync")

	cobraCmd.Flags().StringVar(&cmd.flagSchedule, "schedule", "",
		"Sync periodically using CRON schedule. If not specified, runs only once.")

	return cobraCmd
}

type syncCmd struct {
	ctx          context.Context
	flgSrcFile   string
	flagDstFile  string
	flagSyncFile string
	flagSchedule string

	source v1alpha1.StoreReader
	target v1alpha1.StoreWriter
	sync   *v1alpha1.SyncJob
}

func (cmd *syncCmd) init() error {
	// Init source
	source, err := initStore(cmd.ctx, cmd.flgSrcFile)
	if err != nil {
		return fmt.Errorf("error initializing source store: %w", err)
	}
	cmd.source = source

	// Init target
	target, err := initStore(cmd.ctx, cmd.flagDstFile)
	if err != nil {
		return fmt.Errorf("error initializing target store: %w", err)
	}
	cmd.target = target

	// Init sync request by loading from file and overriding from cli
	sync, err := loadSyncPlan(cmd.flagSyncFile)
	if err != nil {
		return fmt.Errorf("error loading sync plan: %w", err)
	}
	cmd.sync = sync

	if cmd.flagSchedule != "" {
		cmd.sync.Schedule = cmd.flagSchedule
	}

	return nil
}

func (cmd *syncCmd) run(syncReq *v1alpha1.SyncJob) error {
	// Run once
	if syncReq.GetSchedule(cmd.ctx) == nil {
		resp, err := storesync.Sync(cmd.ctx, cmd.source, cmd.target, syncReq.Sync)
		if err != nil {
			return err
		}
		slog.InfoContext(cmd.ctx, resp.Status)

		return nil
	}

	// Run on schedule
	cronTicker, err := cronticker.NewTicker(syncReq.Schedule)
	if err != nil {
		return err
	}
	defer cronTicker.Stop()

	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, os.Interrupt)
	for {
		select {
		case <-cronTicker.C:
			slog.InfoContext(cmd.ctx, "Handling a new sync request...")

			resp, err := storesync.Sync(cmd.ctx, cmd.source, cmd.target, syncReq.Sync)
			if err != nil {
				return err
			}
			slog.InfoContext(cmd.ctx, resp.Status)

		case <-cancel:
			return nil
		}
	}
}

func initStore(ctx context.Context, path string) (v1alpha1.StoreClient, error) {
	store, err := loadStore(path)
	if err != nil {
		return nil, fmt.Errorf("error loading store: %w", err)
	}

	client, err := provider.NewClient(ctx, store)
	if err != nil {
		return nil, fmt.Errorf("error creating client: %w", err)
	}

	return client, nil
}

func loadStore(path string) (*v1alpha1.SecretStoreSpec, error) {
	// Load file
	yamlBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Unmarshal (convert YAML to JSON)
	var storeConfig = struct {
		SecretsStore v1alpha1.SecretStoreSpec `json:"secretsStore"`
	}{}

	jsonBytes, err := yaml.YAMLToJSON(yamlBytes)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(jsonBytes, &storeConfig); err != nil {
		return nil, err
	}

	return &storeConfig.SecretsStore, nil
}

func loadSyncPlan(path string) (*v1alpha1.SyncJob, error) {
	// Load file
	yamlBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Unmarshal (convert YAML to JSON)
	var ruleCfg v1alpha1.SyncJob
	jsonBytes, err := yaml.YAMLToJSON(yamlBytes)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(jsonBytes, &ruleCfg); err != nil {
		return nil, err
	}

	return &ruleCfg, nil
}
