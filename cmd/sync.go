package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/bank-vaults/secret-sync/pkg/apis"
	"github.com/bank-vaults/secret-sync/pkg/sync"
	"github.com/spf13/cobra"
	"os"
	"time"
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
			return cmd.run()
		},
	}

	// Register cmd flags
	cobraCmd.Flags().StringVar(&cmd.flgSrcFile, "source", "", "Source store config file")
	_ = cobraCmd.MarkFlagRequired("source")
	cobraCmd.Flags().StringVar(&cmd.flagDstFile, "dest", "", "Destination store config file")
	_ = cobraCmd.MarkFlagRequired("dest")
	cobraCmd.Flags().StringVar(&cmd.flagRuleFile, "rule", "", "Rule file containing keys and filters. Not needed if --key or --filter used")

	cobraCmd.Flags().StringSliceVar(&cmd.flagKeys, "key", []string{}, "Key to sync. Can specify multiple. Rule file must be empty.")
	cobraCmd.Flags().StringSliceVar(&cmd.flagFilters, "filter", []string{}, "Regex filter for source list keys. Can specify multiple. Rule file must be empty.")

	cobraCmd.Flags().DurationVar(&cmd.flagPeriod, "period", apis.DefaultSyncRequestPeriod, "Synchronization period")
	cobraCmd.Flags().BoolVar(&cmd.flagOnce, "once", false, "Synchronization once and exit")

	return cobraCmd
}

type syncCmd struct {
	flagKeys     []string
	flagFilters  []string
	flgSrcFile   string
	flagDstFile  string
	flagRuleFile string
	flagPeriod   time.Duration
	flagOnce     bool

	source  *apis.SecretStoreSpec
	dest    *apis.SecretStoreSpec
	ruleCfg *ruleConfig
}

func (cmd *syncCmd) init() error {
	var err error

	// Init source
	cmd.source, err = loadStoreSpec(cmd.flgSrcFile)
	if err != nil {
		return err
	}

	// Init dest
	cmd.dest, err = loadStoreSpec(cmd.flagDstFile)
	if err != nil {
		return err
	}

	// Init rule config
	cmd.ruleCfg = &ruleConfig{
		Keys:        cmd.flagKeys,
		ListFilters: cmd.flagFilters,
	}
	if cmd.flagRuleFile != "" {
		cmd.ruleCfg, err = loadRuleSpecs(cmd.flagRuleFile)
		if err != nil {
			return err
		}
	}

	return nil
}

func (cmd *syncCmd) run() error {
	// Start sync
	mgr, err := sync.HandleSync(apis.SyncSecretStoreSpec{
		SourceStore:  cmd.source,
		DestStore:    cmd.dest,
		Keys:         keysToStoreKeys(cmd.ruleCfg.Keys),
		KeyFilters:   cmd.ruleCfg.ListFilters,
		SyncTemplate: cmd.ruleCfg.Template,
		SyncPeriod:   cmd.flagPeriod.String(),
		SyncOnce:     cmd.flagOnce,
	})
	if err != nil {
		return err
	}

	// Wait
	mgr.Wait()

	return nil
}

type ruleConfig struct {
	Keys        []string `json:"keys"`
	ListFilters []string `json:"listFilters"`
	Template    string   `json:"template"`
}

func loadRuleSpecs(path string) (*ruleConfig, error) {
	// Load file
	ruleBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Unmarshal
	var ruleCfg ruleConfig
	if err := json.Unmarshal(ruleBytes, &ruleCfg); err != nil {
		return nil, err
	}
	return &ruleCfg, nil
}

func loadStoreSpec(path string) (*apis.SecretStoreSpec, error) {
	// Load file
	specBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Unmarshal
	var spec apis.SecretStoreSpec
	if err := json.Unmarshal(specBytes, &spec); err != nil {
		return nil, fmt.Errorf("could not unmarshal file %s: %w", path, err)
	}
	return &spec, nil
}

func keysToStoreKeys(keys []string) []apis.StoreKey {
	result := make([]apis.StoreKey, 0)
	for _, key := range keys {
		result = append(result, apis.StoreKey{
			Key: key,
		})
	}
	return result
}
