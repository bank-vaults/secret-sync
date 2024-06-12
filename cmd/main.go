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

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"slices"

	slogmulti "github.com/samber/slog-multi"
	slogsyslog "github.com/samber/slog-syslog"

	"github.com/bank-vaults/secret-sync/pkg/config"
)

func main() {
	config, err := config.LoadConfig()
	if err != nil {
		slog.Error(fmt.Errorf("error loading config: %w", err).Error())
	}

	initLogger(config)

	if err := NewSyncCmd().Execute(); err != nil {
		slog.Error(fmt.Errorf("error executing command: %w", err).Error())
	}
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
