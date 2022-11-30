// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

var (
	clientFromEnv     *client.Client
	clientFromEnvOnce sync.Once
	errClientFromEnv  error
)

func ClientFromEnv() (*client.Client, error) {
	clientFromEnvOnce.Do(func() {
		var err error
		clientFromEnv, err = client.NewClientWithOpts(
			client.FromEnv,
			client.WithAPIVersionNegotiation(),
		)
		if err != nil {
			errClientFromEnv = fmt.Errorf(
				"failed to create Docker client from environment: %w",
				err,
			)
		}
	})

	return clientFromEnv, errClientFromEnv
}

//nolint:revive // Complex but only used in tests.
func RunContainerInBackground(
	ctx context.Context,
	containerName string,
	containerCfg *container.Config,
	hostCfg *container.HostConfig,
) (containerInspect types.ContainerJSON, cleanup func(context.Context) error, err error) {
	dClient, err := ClientFromEnv()
	if err != nil {
		return types.ContainerJSON{}, nil, err
	}

	if hostCfg.NetworkMode.IsUserDefined() {
		_, err = dClient.NetworkInspect(
			ctx,
			hostCfg.NetworkMode.NetworkName(),
			types.NetworkInspectOptions{},
		)
		if client.IsErrNotFound(err) {
			_, err = dClient.NetworkCreate(
				ctx,
				hostCfg.NetworkMode.NetworkName(),
				types.NetworkCreate{
					Driver: "bridge",
					Options: map[string]string{
						"com.docker.network.bridge.enable_ip_masquerade": "true",
						"com.docker.network.driver.mtu":                  "1500",
					},
				},
			)
			if err != nil {
				return types.ContainerJSON{}, nil, fmt.Errorf(
					"failed to create Docker network %q for container: %w",
					hostCfg.NetworkMode.NetworkName(),
					err,
				)
			}
		}
	}

	out, err := dClient.ImagePull(ctx, containerCfg.Image, types.ImagePullOptions{})
	if err != nil {
		return types.ContainerJSON{}, nil, fmt.Errorf(
			"failed to pull image %q: %w",
			containerCfg.Image,
			err,
		)
	}
	defer out.Close()
	_, _ = io.Copy(io.Discard, out)

	created, err := dClient.ContainerCreate(ctx, containerCfg, hostCfg, nil, nil, containerName)
	if err != nil {
		return types.ContainerJSON{}, nil, fmt.Errorf("failed to create container: %w", err)
	}
	containerID := created.ID

	cleanup = func(ctx context.Context) error {
		return ForceDeleteContainer(ctx, containerID)
	}

	err = dClient.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
	if err != nil {
		if deleteErr := cleanup(ctx); deleteErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "WARNING: failed to delete container: %v", deleteErr)
		}

		return types.ContainerJSON{}, nil, fmt.Errorf("failed to start container: %w", err)
	}

	containerInspect, err = dClient.ContainerInspect(ctx, containerID)
	if err != nil {
		if deleteErr := cleanup(ctx); deleteErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "WARNING: failed to delete container: %v", deleteErr)
		}

		return types.ContainerJSON{}, nil, fmt.Errorf(
			"failed to inspect started container: %w",
			err,
		)
	}

	return containerInspect, cleanup, nil
}

func ForceDeleteContainer(ctx context.Context, containerID string) error {
	dClient, err := ClientFromEnv()
	if err != nil {
		return err
	}
	err = dClient.ContainerRemove(
		ctx,
		containerID,
		types.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to delete container: %w", err)
	}
	return nil
}
