// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/docker/go-connections/nat"
	"github.com/foomo/htpasswd"
	g "github.com/onsi/ginkgo/v2"
	gm "github.com/onsi/gomega"
	"github.com/sethvargo/go-password/password"

	"github.com/mesosphere/dynamic-credential-provider/test/e2e/docker"
	"github.com/mesosphere/dynamic-credential-provider/test/e2e/env"
	"github.com/mesosphere/dynamic-credential-provider/test/e2e/seedrng"
	"github.com/mesosphere/dynamic-credential-provider/test/e2e/tls"
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

func (r *Registry) Delete(ctx context.Context) {
	gm.Expect(r.cleanup(ctx)).To(gm.Succeed())
}

func NewRegistry(ctx context.Context, opts ...Opt) *Registry {
	seedrng.EnsureSeeded()

	rOpt := defaultRegistryOptions()
	for _, o := range opts {
		o(&rOpt)
	}

	containerName := strings.ReplaceAll(namesgenerator.GetRandomName(0), "_", "-")

	tlsCertsDir := tls.GenerateCertificates(containerName)

	htpasswdFile, username, passwd := createHtpasswdFile()

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

	containerInspect := docker.RunContainerInBackground(
		ctx,
		containerName,
		&containerCfg,
		&hostCfg,
		env.DockerHubUsername(),
		env.DockerHubPassword(),
	)

	publishedPort, ok := containerInspect.NetworkSettings.Ports[nat.Port("5000/tcp")]
	gm.Expect(ok).To(gm.BeTrue())

	return &Registry{
		username:        username,
		password:        passwd,
		address:         net.JoinHostPort(containerName, "5000"),
		hostPortAddress: net.JoinHostPort(publishedPort[0].HostIP, publishedPort[0].HostPort),
		caCertFile:      filepath.Join(tlsCertsDir, "ca.crt"),
	}
}

//nolint:revive // 5 return values is OK for this test function.
func createHtpasswdFile() (htpasswdFile, username, passwd string) {
	dir, err := os.MkdirTemp("", "registry-auth-*")
	gm.Expect(err).NotTo(gm.HaveOccurred())
	g.DeferCleanup(os.RemoveAll, dir)

	f := filepath.Join(dir, "htpasswd")
	username = namesgenerator.GetRandomName(0)
	passwd, err = password.Generate(
		32,
		8,
		8,
		false,
		false,
	) //nolint:revive // Constants for password generation.
	gm.Expect(err).NotTo(gm.HaveOccurred())

	gm.Expect(htpasswd.SetPassword(f, username, passwd, htpasswd.HashBCrypt)).To(gm.Succeed())

	return f, username, passwd
}
