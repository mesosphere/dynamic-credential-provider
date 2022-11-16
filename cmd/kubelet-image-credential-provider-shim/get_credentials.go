// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mesosphere/kubelet-image-credential-provider-shim/pkg/credentialprovider/plugin"
	"github.com/mesosphere/kubelet-image-credential-provider-shim/pkg/credentialprovider/shim"
)

type getCredentialsOptions struct {
	configFile string
}

func (o *getCredentialsOptions) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(
		&o.configFile, "config", "c", o.configFile,
		"path to the configuration file for the Kubelet credential provider shim",
	)
}

func defaultCredentialsOptions() *getCredentialsOptions {
	return &getCredentialsOptions{
		configFile: "/etc/kubernetes/image-credential-provider/kubelet-image-credential-provider-shim.yaml",
	}
}

func newGetCredentialsCmd() *cobra.Command {
	opts := defaultCredentialsOptions()

	cmd := &cobra.Command{
		Use:   "get-credentials",
		Short: "Get authentication credentials",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			provider, err := shim.NewProviderFromConfigFile(opts.configFile)
			if err != nil {
				return fmt.Errorf("error initializing shim credential provider: %w", err)
			}

			err = plugin.NewProvider(provider).Run(context.Background())
			if err != nil {
				return fmt.Errorf("error running shim credential provider: %w", err)
			}

			return nil
		},
	}
	opts.AddFlags(cmd)

	return cmd
}
