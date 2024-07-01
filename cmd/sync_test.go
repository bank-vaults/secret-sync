// Copyright © 2023 Bank-Vaults Maintainers
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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const secretStoreTemplate = `
secretsStore:
  local:
    storePath: %q
`

// TODO: Expand tests
func TestSync(t *testing.T) {
	tests := []struct {
		name   string
		source string
		target string
		sync   string
	}{
		{
			name:   "Sync from local-store to local-store",
			source: localStore(t, "testdata"),
			target: localStore(t, filepath.Join(os.TempDir(), "target")),
			sync:   "testdata/syncjob.yaml",
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			syncCmd.SetArgs([]string{
				"--source", ttp.source,
				"--target", ttp.target,
				"--sync", ttp.sync,
			})

			err := syncCmd.ExecuteContext(context.Background())
			require.NoError(t, err, "Unexpected error")
		})
	}
}

func localStore(t *testing.T, dirPath string) string {
	// Ensure dir exists
	path, err := filepath.Abs(dirPath)
	require.NoError(t, err)

	// Create file
	tmpFile, err := os.CreateTemp("", "source.*")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Remove(tmpFile.Name())
	})

	// Write
	_, err = tmpFile.WriteString(fmt.Sprintf(secretStoreTemplate, path))
	require.NoError(t, err)

	return tmpFile.Name()
}
