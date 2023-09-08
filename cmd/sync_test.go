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

func TestSync(t *testing.T) {
	syncCmd := NewSyncCmd()
	syncCmd.SetArgs([]string{
		"--source", storeFile(t, "testdata/source"),
		"--dest", storeFile(t, filepath.Join(os.TempDir(), "dest")),
		"--sync", "testdata/syncjob.yaml",
		"--once",
	})
	err := syncCmd.Execute()
	assert.Nil(t, err)
}

func storeFile(t *testing.T, dirPath string) string {
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
permissions: ReadWrite
provider:
  file:
    dir-path: %q
`, path)))
	assert.Nil(t, err)

	return tmpFile.Name()
}
