package cmd

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSync(t *testing.T) {
	syncCmd := NewSyncCmd()
	syncCmd.SetArgs([]string{
		"--source", "testdata/store-file-source.yaml",
		"--dest", "testdata/store-file-dest.yaml",
		"--sync", "testdata/syncjob.yaml",
		"--once",
	})
	err := syncCmd.Execute()
	assert.Nil(t, err)
}
