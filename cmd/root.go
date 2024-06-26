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
	"net"
	"os"
	"slices"

	slogmulti "github.com/samber/slog-multi"
	slogsyslog "github.com/samber/slog-syslog"
	"github.com/spf13/cobra"

	"github.com/bank-vaults/secret-sync/pkg/config"
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
		initLogger(config.LoadConfig())
	})
}

func initLogger(config *config.Config) {
	var level slog.Level

	err := level.UnmarshalText([]byte(config.LogLevel))
	if err != nil { // Silently fall back to info level
		level = slog.LevelInfo
	}

	levelFilter := func(levels ...slog.Level) func(ctx context.Context, r slog.Record) bool {
		return func(_ context.Context, r slog.Record) bool {
			return slices.Contains(levels, r.Level)
		}
	}

	router := slogmulti.Router()

	if config.JSONLog {
		// Send logs with level higher than warning to stderr
		router = router.Add(
			slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}),
			levelFilter(slog.LevelWarn, slog.LevelError),
		)

		// Send info and debug logs to stdout
		router = router.Add(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
			levelFilter(slog.LevelDebug, slog.LevelInfo),
		)
	} else {
		// Send logs with level higher than warning to stderr
		router = router.Add(
			slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}),
			levelFilter(slog.LevelWarn, slog.LevelError),
		)

		// Send info and debug logs to stdout
		router = router.Add(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
			levelFilter(slog.LevelDebug, slog.LevelInfo),
		)
	}

	if config.LogServer != "" {
		writer, err := net.Dial("udp", config.LogServer)

		// We silently ignore syslog connection errors for the lack of a better solution
		if err == nil {
			router = router.Add(slogsyslog.Option{Level: slog.LevelInfo, Writer: writer}.NewSyslogHandler())
		}
	}

	// TODO: add level filter handler
	logger := slog.New(router.Handler())
	logger = logger.With(slog.String("app", "secret-sync"))

	// Set the default logger to the configured logger,
	// enabling direct usage of the slog package for logging.
	slog.SetDefault(logger)
}
