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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TODO: Expand tests

func TestSync(t *testing.T) {
	syncCmd := NewSyncCmd()
	syncCmd.SetArgs([]string{
		"--source", localStore(t, "testdata"),
		"--target", localStore(t, filepath.Join(os.TempDir(), "target")),
		"--sync", "testdata/syncjob.yaml",
	})
	err := syncCmd.Execute()
	assert.Nil(t, err)
}

func localStore(t *testing.T, dirPath string) string {
	// Ensure dir exists
	path, err := filepath.Abs(dirPath)
	assert.Nil(t, err)

	// Create file
	tmpFile, err := os.CreateTemp("", "source.*")
	assert.Nil(t, err)
	t.Cleanup(func() {
		_ = os.Remove(tmpFile.Name())
	})

	// Write
	_, err = tmpFile.Write([]byte(fmt.Sprintf(`
secretsStore:
  local:
    storePath: %q
`, path)))
	assert.Nil(t, err)

	return tmpFile.Name()
}
