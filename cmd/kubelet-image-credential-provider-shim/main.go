// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	logger := logrus.New()

	rootCmd := &cobra.Command{
		Use:          filepath.Base(os.Args[0]),
		SilenceUsage: true,
	}

	rootCmd.AddCommand(newInstallCmd(logger))

	if err := rootCmd.Execute(); err != nil {
		logger.Fatal(err)
	}
}
