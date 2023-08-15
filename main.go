package main

import (
	"github.com/bank-vaults/secret-sync/cmd"
	"github.com/sirupsen/logrus"
)

func main() {
	if err := cmd.NewSyncCmd().Execute(); err != nil {
		logrus.Fatalf("error executing command: %v", err)
	}
}
