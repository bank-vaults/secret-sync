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
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/bank-vaults/secret-sync/pkg/config"
	"github.com/bank-vaults/secret-sync/pkg/utils"
)

var rootCmd = &cobra.Command{
	Use: "secret-sync",
	Long: `Secret Sync exposes a generic way to interact with external secret storage systems
like HashiCorp Vault and provides a set of API models
to interact and orchestrate the synchronization of secrets between them.`,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

func Execute() {
	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		slog.ErrorContext(rootCmd.Context(), fmt.Sprintf("failed to execute command: %v", err))
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(func() {
		utils.InitLogger(config.LoadConfig())
	})
}
