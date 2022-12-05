// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"go.uber.org/multierr"
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
	pullUsername, pullPassword string,
) (types.ContainerJSON, error) {
	dClient, err := ClientFromEnv()
	if err != nil {
		return types.ContainerJSON{}, fmt.Errorf("failed to create Docker client: %w", err)
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
		}
		if err != nil {
			return types.ContainerJSON{}, fmt.Errorf("failed to create Docker network: %w", err)
		}
	}

	out, err := dClient.ImagePull(ctx, containerCfg.Image, types.ImagePullOptions{})
	defer func() { _ = out.Close() }()
	if err != nil {
		_, _ = io.Copy(os.Stderr, out)
		return types.ContainerJSON{}, fmt.Errorf("failed to pull Docker image: %w", err)
	}
	_, _ = io.Copy(io.Discard, out)

	created, err := dClient.ContainerCreate(ctx, containerCfg, hostCfg, nil, nil, containerName)
	if err != nil {
		return types.ContainerJSON{}, fmt.Errorf("failed to create container: %w", err)
	}
	containerID := created.ID

	if err := dClient.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		//nolint:contextcheck // Best effort background deletion.
		if deleteErr := ForceDeleteContainer(context.Background(), containerID); deleteErr != nil {
			err = multierr.Combine(err, deleteErr)
		}
		return types.ContainerJSON{}, fmt.Errorf("failed to start Docker container: %w", err)
	}

	containerInspect, err := dClient.ContainerInspect(ctx, containerID)
	if err != nil {
		//nolint:contextcheck // Best effort background deletion.
		if deleteErr := ForceDeleteContainer(context.Background(), containerID); deleteErr != nil {
			err = multierr.Combine(err, deleteErr)
		}
		return types.ContainerJSON{}, fmt.Errorf(
			"failed to inspect started Docker container: %w",
			err,
		)
	}

	return containerInspect, nil
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

func RetagAndPushImage( //nolint:revive // Lots of args is fine in these tests.
	ctx context.Context,
	srcImage, destImage string,
	pullUsername, pullPassword string,
	pushUsername, pushPassword string,
) error {
	dClient, err := ClientFromEnv()
	if err != nil {
		return err
	}

	out, err := dClient.ImagePull(
		ctx,
		srcImage,
		types.ImagePullOptions{RegistryAuth: authString(pullUsername, pullPassword)},
	)
	defer func() { _ = out.Close() }()
	if err != nil {
		_, _ = io.Copy(os.Stderr, out)
		return fmt.Errorf(
			"failed to pull image %q: %w",
			srcImage,
			err,
		)
	}
	_, _ = io.Copy(io.Discard, out)

	if err := dClient.ImageTag(ctx, srcImage, destImage); err != nil {
		return fmt.Errorf("failed to retag image: %w", err)
	}
	defer func() { _, _ = dClient.ImageRemove(ctx, destImage, types.ImageRemoveOptions{}) }()

	out, err = dClient.ImagePush(
		ctx,
		destImage,
		types.ImagePushOptions{RegistryAuth: authString(pushUsername, pushPassword)},
	)
	defer func() { _ = out.Close() }()
	if err != nil {
		_, _ = io.Copy(os.Stderr, out)
		return fmt.Errorf("failed to push retagged image: %w", err)
	}
	_, _ = io.Copy(io.Discard, out)

	return nil
}

func authString(username, password string) string {
	authConfig := types.AuthConfig{
		Username: username,
		Password: password,
	}
	encodedJSON, _ := json.Marshal(authConfig)

	return base64.URLEncoding.EncodeToString(encodedJSON)
}
