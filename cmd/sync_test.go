package cmd

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
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

	return tmpFile.Name()
}
