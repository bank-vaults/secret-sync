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

package config

import (
	"os"

	"github.com/spf13/cast"
)

const (
	LogLevelEnv  = "SECRET_SYNC_LOG_LEVEL"
	JSONLogEnv   = "SECRET_SYNC_JSON_LOG"
	LogServerEnv = "SECRET_SYNC_LOG_SERVER"
)

type Config struct {
	LogLevel  string `json:"log_level"`
	JSONLog   bool   `json:"json_log"`
	LogServer string `json:"log_server"`
}

func LoadConfig() *Config {
	return &Config{
		LogLevel:  os.Getenv(LogLevelEnv),
		JSONLog:   cast.ToBool(os.Getenv(JSONLogEnv)),
		LogServer: os.Getenv(LogServerEnv),
	}
}
