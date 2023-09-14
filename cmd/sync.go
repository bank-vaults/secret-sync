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

package cmd

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"

	"github.com/ghodss/yaml"
	"github.com/krayzpipes/cronticker/cronticker"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/bank-vaults/secret-sync/pkg/apis/v1alpha1"
	"github.com/bank-vaults/secret-sync/pkg/provider"
	"github.com/bank-vaults/secret-sync/pkg/storesync"
)

func NewSyncCmd() *cobra.Command {
	// Create cmd
	cmd := &syncCmd{}
	cobraCmd := &cobra.Command{
		Use:   "sync",
		Short: "Synchronizes secrets from a source to a target store based on sync strategy.",
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := cmd.init(); err != nil {
				return err
			}
			return cmd.run(cmd.sync)
		},
	}

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

	cobraCmd.Flags().StringVar(&cmd.flagSchedule, "schedule", v1alpha1.DefaultSyncJobSchedule,
		"Sync on CRON schedule. Either --schedule or --once should be specified.")
	cobraCmd.Flags().BoolVar(&cmd.flagOnce, "once", false,
		"Synchronize once and exit. Either --schedule or --once should be specified.")

	return cobraCmd
}

type syncCmd struct {
	flgSrcFile   string
	flagDstFile  string
	flagSyncFile string
	flagSchedule string
	flagOnce     bool

	source v1alpha1.StoreReader
	target v1alpha1.StoreWriter
	sync   *v1alpha1.SyncJob
}

func (cmd *syncCmd) init() error {
	var err error

	// Init source
	srcStore, err := loadStore(cmd.flgSrcFile)
	if err != nil {
		return err
	}
	cmd.source, err = provider.NewClient(context.Background(), srcStore)
	if err != nil {
		return err
	}

	// Init target
	targetStore, err := loadStore(cmd.flagDstFile)
	if err != nil {
		return err
	}
	cmd.target, err = provider.NewClient(context.Background(), targetStore)
	if err != nil {
		return err
	}

	// Init sync request by loading from file and overriding from cli
	cmd.sync, err = loadStrategy(cmd.flagSyncFile)
	if err != nil {
		return err
	}
	if cmd.flagOnce {
		cmd.sync.RunOnce = cmd.flagOnce
	}
	if cmd.flagSchedule != "" {
		cmd.sync.Schedule = cmd.flagSchedule
	}

	return nil
}

func (cmd *syncCmd) run(syncReq *v1alpha1.SyncJob) error {
	// Run once
	if syncReq.RunOnce {
		resp, err := storesync.Sync(context.Background(), cmd.source, cmd.target, syncReq.Sync)
		if err != nil {
			return err
		}
		logrus.Info(resp.Status)
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
			logrus.Info("Handling a new sync request...")
			resp, err := storesync.Sync(context.Background(), cmd.source, cmd.target, syncReq.Sync)
			if err != nil {
				return err
			}
			logrus.Info(resp.Status)

		case <-cancel:
			return nil
		}
	}
}

func loadStrategy(path string) (*v1alpha1.SyncJob, error) {
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

func loadStore(path string) (*v1alpha1.ProviderBackend, error) {
	// Load file
	yamlBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Unmarshal (convert YAML to JSON)
	var spec v1alpha1.ProviderBackend
	jsonBytes, err := yaml.YAMLToJSON(yamlBytes)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(jsonBytes, &spec); err != nil {
		return nil, err
	}
	return &spec, nil
}
