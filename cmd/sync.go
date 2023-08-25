package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bank-vaults/secret-sync/pkg/apis/v1alpha1"
	"github.com/bank-vaults/secret-sync/pkg/provider"
	"github.com/bank-vaults/secret-sync/pkg/storesync"
	"github.com/ghodss/yaml"
	"github.com/krayzpipes/cronticker/cronticker"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
)

func NewSyncCmd() *cobra.Command {
	// Create cmd
	cmd := &syncCmd{}
	cobraCmd := &cobra.Command{
		Use:   "sync",
		Short: "Synchronizes a key-value destination store from source store",
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := cmd.init(); err != nil {
				return err
			}
			return cmd.run(cmd.sync)
		},
	}

	// Register cmd flags
	cobraCmd.Flags().StringVar(&cmd.flgSrcFile, "source", "", "Source store config file")
	_ = cobraCmd.MarkFlagRequired("source")
	cobraCmd.Flags().StringVar(&cmd.flagDstFile, "dest", "", "Destination store config file")
	_ = cobraCmd.MarkFlagRequired("dest")
	cobraCmd.Flags().StringVar(&cmd.flagSyncFile, "sync", "", "Sync job config file")

	cobraCmd.Flags().StringSliceVar(&cmd.flagKeys, "key", []string{}, "Key to sync. Can specify multiple. Overrides --sync params")
	cobraCmd.Flags().StringSliceVar(&cmd.flagFilters, "filter", []string{}, "Regex filter for source list keys. Can specify multiple. Overrides --sync params")
	cobraCmd.Flags().StringVar(&cmd.flagTemplate, "template", "", "Conversion template to use. Overrides --sync params")
	cobraCmd.Flags().StringVar(&cmd.flagSchedule, "schedule", v1alpha1.DefaultSyncJobSchedule, "Synchronization CRON schedule. Overrides --sync params")
	cobraCmd.Flags().BoolVar(&cmd.flagOnce, "once", false, "Synchronize once and exit. Overrides --sync params")

	return cobraCmd
}

type syncCmd struct {
	flagKeys     []string
	flagFilters  []string
	flgSrcFile   string
	flagDstFile  string
	flagSyncFile string
	flagTemplate string
	flagSchedule string
	flagOnce     bool

	source v1alpha1.StoreReader
	dest   v1alpha1.StoreWriter
	sync   v1alpha1.SyncJobSpec
}

func (cmd *syncCmd) init() error {
	var err error

	// Init source
	srcStore, err := loadStore(cmd.flgSrcFile)
	if err != nil {
		return err
	}
	if !srcStore.GetPermissions().CanPerform(v1alpha1.SecretStorePermissionsRead) {
		return fmt.Errorf("source does not have Read permissions")
	}
	cmd.source, err = provider.NewClient(context.Background(), &srcStore.Provider)
	if err != nil {
		return err
	}

	// Init dest
	destStore, err := loadStore(cmd.flagDstFile)
	if err != nil {
		return err
	}
	if !destStore.GetPermissions().CanPerform(v1alpha1.SecretStorePermissionsWrite) {
		return fmt.Errorf("dest does not have Write permissions")
	}
	cmd.dest, err = provider.NewClient(context.Background(), &destStore.Provider)
	if err != nil {
		return err
	}

	// Init sync request by loading from file and overriding from cli
	if cmd.flagSyncFile != "" {
		request, err := loadRequest(cmd.flagSyncFile)
		if err != nil {
			return err
		}
		cmd.sync = *request
	}
	if len(cmd.flagKeys) > 0 {
		cmd.sync.Keys = keysToStoreKeys(cmd.flagKeys)
	}
	if len(cmd.flagFilters) > 0 {
		cmd.sync.ListFilters = cmd.flagFilters
	}
	if cmd.flagTemplate != "" {
		cmd.sync.Template = cmd.flagTemplate
	}
	if cmd.flagOnce {
		cmd.sync.RunOnce = cmd.flagOnce
	}
	if cmd.flagSchedule != "" {
		cmd.sync.Schedule = cmd.flagSchedule
	}

	return nil
}

func (cmd *syncCmd) run(syncReq v1alpha1.SyncJobSpec) error {
	// Create request
	request := storesync.Request{
		Source:      cmd.source,
		Dest:        cmd.dest,
		Keys:        syncReq.Keys,
		ListFilters: syncReq.GetListFilters(),
		Converter:   syncReq.ConvertKey,
	}

	// Run once
	if syncReq.RunOnce {
		resp, err := storesync.Sync(context.Background(), request)
		if resp != nil {
			logrus.Info(resp.Status)
		}
		return err
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
			resp, _ := storesync.Sync(context.Background(), request)
			if resp != nil {
				logrus.Info(resp.Status)
			}

		case <-cancel:
			return nil
		}
	}
}

// loadRequest loads apis.SyncJobSpec data from a YAML file.
func loadRequest(path string) (*v1alpha1.SyncJobSpec, error) {
	// Load file
	yamlBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Unmarshal (convert YAML to JSON)
	var ruleCfg v1alpha1.SyncJobSpec
	jsonBytes, err := yaml.YAMLToJSON(yamlBytes)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(jsonBytes, &ruleCfg); err != nil {
		return nil, err
	}
	return &ruleCfg, nil
}

// loadStore loads apis.SecretStoreSpec from a YAML file.
func loadStore(path string) (*v1alpha1.SecretStoreSpec, error) {
	// Load file
	yamlBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Unmarshal (convert YAML to JSON)
	var spec v1alpha1.SecretStoreSpec
	jsonBytes, err := yaml.YAMLToJSON(yamlBytes)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(jsonBytes, &spec); err != nil {
		return nil, err
	}
	return &spec, nil
}

func keysToStoreKeys(keys []string) []v1alpha1.StoreKey {
	result := make([]v1alpha1.StoreKey, 0)
	for _, key := range keys {
		result = append(result, v1alpha1.StoreKey{
			Key: key,
		})
	}
	return result
}
