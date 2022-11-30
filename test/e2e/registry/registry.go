// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/docker/go-connections/nat"
	"github.com/foomo/htpasswd"
	"github.com/sethvargo/go-password/password"

	"github.com/mesosphere/kubelet-image-credential-provider-shim/test/e2e/docker"
	"github.com/mesosphere/kubelet-image-credential-provider-shim/test/e2e/env"
	"github.com/mesosphere/kubelet-image-credential-provider-shim/test/e2e/seedrng"
	"github.com/mesosphere/kubelet-image-credential-provider-shim/test/e2e/tls"
)

type registryOptions struct {
	image         string
	dockerNetwork string
}

type Opt func(*registryOptions)

func WithImage(image string) Opt {
	return func(ro *registryOptions) { ro.image = image }
}

func WithDockerNetwork(network string) Opt {
	return func(ro *registryOptions) { ro.dockerNetwork = network }
}

func defaultRegistryOptions() registryOptions {
	return registryOptions{
		image:         "registry:2",
		dockerNetwork: "kind",
	}
}

type Registry struct {
	cleanup func(context.Context) error

	username        string
	password        string
	address         string
	hostPortAddress string
	caCertFile      string
}

func (r *Registry) Username() string {
	return r.username
}

func (r *Registry) Password() string {
	return r.password
}

func (r *Registry) Address() string {
	return r.address
}

func (r *Registry) HostPortAddress() string {
	return r.hostPortAddress
}

func (r *Registry) CACertFile() string {
	return r.caCertFile
}

func (r *Registry) Delete(ctx context.Context) error {
	if err := r.cleanup(ctx); err != nil {
		return fmt.Errorf("failed to delete Docker registry container: %w", err)
	}
	return nil
}

//nolint:revive // Complex function for test is OK.
func NewRegistry(ctx context.Context, opts ...Opt) (*Registry, error) {
	seedrng.EnsureSeeded()

	rOpt := defaultRegistryOptions()
	for _, o := range opts {
		o(&rOpt)
	}

	containerName := strings.ReplaceAll(namesgenerator.GetRandomName(0), "_", "-")

	tlsCertsDir, cleanupTLSCerts, err := tls.GenerateCertificates(containerName)
	if err != nil {
		return nil, err
	}

	htpasswdFile, username, passwd, cleanupHtpasswd, err := createHtpasswdFile()
	if err != nil {
		if cleanupErr := cleanupTLSCerts(); cleanupErr != nil {
			_, _ = fmt.Fprintf(
				os.Stderr,
				"WARNING: failed to cleanup TLS certificates dir: %v",
				cleanupErr,
			)
		}
		return nil, err
	}

	containerCfg := container.Config{
		Image:        rOpt.image,
		ExposedPorts: nat.PortSet{nat.Port("5000"): struct{}{}},
		Env: []string{
			"REGISTRY_AUTH=htpasswd",
			"REGISTRY_AUTH_HTPASSWD_REALM=Registry Realm",
			"REGISTRY_AUTH_HTPASSWD_PATH=/auth/htpasswd",
			"REGISTRY_HTTP_TLS_CERTIFICATE=/certs/tls.crt",
			"REGISTRY_HTTP_TLS_KEY=/certs/tls.key",
		},
	}
	hostCfg := container.HostConfig{
		AutoRemove:   true,
		NetworkMode:  container.NetworkMode(rOpt.dockerNetwork),
		PortBindings: nat.PortMap{nat.Port("5000"): []nat.PortBinding{{HostIP: "127.0.0.1"}}},
		Mounts: []mount.Mount{{
			Type:     mount.TypeBind,
			Source:   htpasswdFile,
			Target:   "/auth/htpasswd",
			ReadOnly: true,
		}, {
			Type:     mount.TypeBind,
			Source:   tlsCertsDir,
			Target:   "/certs",
			ReadOnly: true,
		}},
	}

	const (
		warningTLSDir      = "WARNING: failed to cleanup TLS certificates dir: %v"
		warningHtpasswdDir = "WARNING: failed to cleanup htpasswd dir: %v"
	)

	containerInspect, cleanupContainer, err := docker.RunContainerInBackground(
		ctx,
		containerName,
		&containerCfg,
		&hostCfg,
		env.DockerHubUsername(),
		env.DockerHubPassword(),
	)
	if err != nil {
		if cleanupErr := cleanupTLSCerts(); cleanupErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, warningTLSDir, cleanupErr)
		}
		if cleanupErr := cleanupHtpasswd(); cleanupErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, warningHtpasswdDir, cleanupErr)
		}
		return nil, fmt.Errorf("failed to start Docker registry container: %w", err)
	}

	publishedPort, ok := containerInspect.NetworkSettings.Ports[nat.Port("5000/tcp")]
	if !ok {
		if cleanupErr := cleanupTLSCerts(); cleanupErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, warningTLSDir, cleanupErr)
		}
		if cleanupErr := cleanupHtpasswd(); cleanupErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, warningHtpasswdDir, cleanupErr)
		}
		if cleanupErr := cleanupContainer(ctx); cleanupErr != nil {
			_, _ = fmt.Fprintf(
				os.Stderr,
				"WARNING: failed to delete Docker registry container: %v",
				cleanupErr,
			)
		}
		return nil, errors.New("failed to discover Docker registry port")
	}

	r := &Registry{
		username:        username,
		password:        passwd,
		address:         net.JoinHostPort(containerName, "5000"),
		hostPortAddress: net.JoinHostPort(publishedPort[0].HostIP, publishedPort[0].HostPort),
		caCertFile:      filepath.Join(tlsCertsDir, "ca.crt"),
		cleanup: func(ctx context.Context) error {
			if cleanupErr := cleanupTLSCerts(); cleanupErr != nil {
				_, _ = fmt.Fprintf(os.Stderr, warningTLSDir, cleanupErr)
			}
			if cleanupErr := cleanupHtpasswd(); cleanupErr != nil {
				_, _ = fmt.Fprintf(os.Stderr, warningHtpasswdDir, cleanupErr)
			}
			return cleanupContainer(ctx)
		},
	}

	return r, nil
}

//nolint:revive // 5 return values is OK for this test function.
func createHtpasswdFile() (htpasswdFile, username, passwd string, cleanup func() error, err error) {
	dir, err := os.MkdirTemp("", "registry-auth-*")
	if err != nil {
		return "", "", "", nil, fmt.Errorf(
			"failed to create temporary directory for registry auth: %w",
			err,
		)
	}
	cleanup = func() error { return os.RemoveAll(dir) }

	f := filepath.Join(dir, "htpasswd")
	username = namesgenerator.GetRandomName(0)
	passwd, err = password.Generate(
		32,
		8,
		8,
		false,
		false,
	) //nolint:revive // Constants for password generation.
	if err != nil {
		_ = cleanup()
		return "", "", "", nil, fmt.Errorf("failed to generate password for registry auth: %w", err)
	}

	err = htpasswd.SetPassword(f, username, passwd, htpasswd.HashBCrypt)
	if err != nil {
		_ = cleanup()
		return "", "", "", nil, fmt.Errorf(
			"failed to write password to htpasswd file for registry auth: %w",
			err,
		)
	}

	return f, username, passwd, cleanup, nil
}
