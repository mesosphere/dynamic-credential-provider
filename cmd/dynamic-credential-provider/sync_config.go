// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"path"
	"path/filepath"

	"github.com/otiai10/copy"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"k8s.io/klog/v2"

	"github.com/mesosphere/dynamic-credential-provider/pkg/filewatcher"
)

//nolint:gosec // No credentials here.
const (
	defaultCredentialProviderDir               = "/etc/kubernetes/image-credential-provider"
	defaultDynamicCredentialProviderConfigPath = defaultCredentialProviderDir + "/dynamic-credential-provider-config.yaml"
	defaultStaticCredentialsPath               = defaultCredentialProviderDir + "/static-image-credentials.json"
)

type syncConfigOptions struct {
	filesToSync map[string]string
}

func (o *syncConfigOptions) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringToStringVar(
		&o.filesToSync, "config-file", o.filesToSync,
		"config files to synchronize, watching the source and writing to target",
	)
	_ = cmd.MarkFlagRequired("config-file")
}

func defaultSyncConfigOptionsOptions() *syncConfigOptions {
	return &syncConfigOptions{
		filesToSync: map[string]string{
			defaultDynamicCredentialProviderConfigPath: path.Join(
				"/host",
				defaultDynamicCredentialProviderConfigPath,
			),
			defaultStaticCredentialsPath: path.Join(
				"/host",
				defaultStaticCredentialsPath,
			),
		},
	}
}

func newSyncConfigCmd() *cobra.Command {
	opts := defaultSyncConfigOptionsOptions()

	cmd := &cobra.Command{
		Use:   "sync-config",
		Short: "Synchronize config files, copying specified sources to specified destinations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			g, groupCtx := errgroup.WithContext(ctx)

			for src, dest := range opts.filesToSync {
				g.Go(func() error {
					fw, err := filewatcher.New(src, func(fp string) error {
						lstat, err := os.Lstat(fp)
						if err != nil {
							return err
						}
						if lstat.Mode().Type()&os.ModeSymlink != 0 {
							fp, err = os.Readlink(fp)
							if err != nil {
								return err
							}
							if !filepath.IsAbs(fp) {
								fp = filepath.Join(filepath.Dir(src), fp)
							}
						}
						return copy.Copy(
							fp,
							dest,
						)
					})
					if err != nil {
						return err
					}

					return fw.Start(groupCtx)
				})
			}

			return g.Wait()
		},
	}
	opts.AddFlags(cmd)

	klogFlagSet := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlagSet)
	cmd.Flags().AddGoFlagSet(klogFlagSet)

	return cmd
}
