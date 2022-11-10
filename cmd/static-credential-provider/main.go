// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/mesosphere/kubelet-image-credential-provider-shim/pkg/credentialprovider"
)

const (
	//nolint:gosec // not credentials
	defaultCredentialsFile = "/etc/kubernetes/image-credentials.json"
)

func main() {
	logger := logrus.New()

	rootCmd := &cobra.Command{
		Use:   "static-credential-provider <credentials-file>",
		Short: "Static credential provider for Kubelet",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			credentialsFile := defaultCredentialsFile
			if len(args) == 1 {
				credentialsFile = args[0]
			}
			provider, err := credentialprovider.NewStaticProvider(credentialsFile)
			if err != nil {
				return fmt.Errorf("error initializing static credential provider: %w", err)
			}

			err = credentialprovider.NewCredentialProvider(provider).Run(context.TODO())
			if err != nil {
				return fmt.Errorf("error running static credential provider: %w", err)
			}

			return nil
		},
	}

	if err := rootCmd.Execute(); err != nil {
		logger.Fatal(err)
	}
}
