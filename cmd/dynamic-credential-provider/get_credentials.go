// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mesosphere/dynamic-credential-provider/pkg/credentialprovider/dynamic"
	"github.com/mesosphere/dynamic-credential-provider/pkg/credentialprovider/plugin"
)

type getCredentialsOptions struct {
	configFile string
}

func (o *getCredentialsOptions) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(
		&o.configFile, "config", "c", o.configFile,
		"path to the configuration file for the dynamic Kubelet credential provider",
	)
}

func defaultCredentialsOptions() *getCredentialsOptions {
	return &getCredentialsOptions{
		configFile: "/etc/kubernetes/image-credential-provider/dynamic-credential-provider.yaml",
	}
}

func newGetCredentialsCmd() *cobra.Command {
	opts := defaultCredentialsOptions()

	cmd := &cobra.Command{
		Use:   "get-credentials",
		Short: "Get authentication credentials",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			provider, err := dynamic.NewProviderFromConfigFile(opts.configFile)
			if err != nil {
				return fmt.Errorf("error initializing dynamic credential provider: %w", err)
			}

			err = plugin.NewProvider(provider).Run(context.Background())
			if err != nil {
				return fmt.Errorf("error running dynamic credential provider: %w", err)
			}

			return nil
		},
	}
	opts.AddFlags(cmd)

	return cmd
}
