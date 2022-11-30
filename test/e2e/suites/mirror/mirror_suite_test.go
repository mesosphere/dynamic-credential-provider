// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package mirror_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"text/template"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/otiai10/copy"
	"sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
	kindcluster "sigs.k8s.io/kind/pkg/cluster"

	"github.com/mesosphere/kubelet-image-credential-provider-shim/test/e2e/cluster"
	"github.com/mesosphere/kubelet-image-credential-provider-shim/test/e2e/registry"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mirror Suite")
}

var (
	kindCluster    cluster.Cluster
	mirrorRegistry *registry.Registry
)

//nolint:gosec // No credentials here.
const (
	kubeadmInitPatchKubeletCredentialProviderExtraArgs = `kind: InitConfiguration
nodeRegistration:
  kubeletExtraArgs:
    image-credential-provider-config: /etc/kubernetes/image-credential-provider/image-credential-provider-config.yaml
    image-credential-provider-bin-dir: /etc/kubernetes/image-credential-provider/`

	kubeadmJoinPatchKubeletCredentialProviderExtraArgs = `kind: JoinConfiguration
nodeRegistration:
  kubeletExtraArgs:
    image-credential-provider-config: /etc/kubernetes/image-credential-provider/image-credential-provider-config.yaml
    image-credential-provider-bin-dir: /etc/kubernetes/image-credential-provider/`
)

func testdataPath(f string) string {
	return filepath.Join("testdata", f)
}

var _ = SynchronizedBeforeSuite(
	func(ctx SpecContext) {
		By("Starting Docker registry")
		r, err := registry.NewRegistry(ctx)
		Expect(err).ShouldNot(HaveOccurred())
		DeferCleanup(func(ctx SpecContext) error {
			return r.Delete(ctx)
		}, NodeTimeout(time.Second))
		mirrorRegistry = r

		By("Setting up kubelet credential providers")
		providerBinDir := GinkgoT().TempDir()
		configTmpl := template.Must(template.ParseGlob(testdataPath("*.tmpl")))
		templatedFile, err := os.Create(
			filepath.Join(providerBinDir, "image-credential-provider-config.yaml"),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(configTmpl.ExecuteTemplate(
			templatedFile,
			"image-credential-provider-config.yaml.tmpl",
			struct{ MirrorAddress string }{MirrorAddress: r.Address()},
		)).To(Succeed())
		templatedFile, err = os.Create(
			filepath.Join(providerBinDir, "shim-credential-provider-config.yaml"),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(configTmpl.ExecuteTemplate(
			templatedFile,
			"shim-credential-provider-config.yaml.tmpl",
			struct{ MirrorAddress string }{MirrorAddress: r.Address()},
		)).To(Succeed())
		templatedFile, err = os.Create(
			filepath.Join(providerBinDir, "static-image-credentials.json"),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(configTmpl.ExecuteTemplate(
			templatedFile,
			"static-image-credentials.json.tmpl",
			struct {
				MirrorAddress, MirrorUsername, MirrorPassword string
			}{
				MirrorAddress:  r.Address(),
				MirrorUsername: r.Username(),
				MirrorPassword: r.Password(),
			},
		)).To(Succeed())
		Expect(templatedFile.Close()).To(Succeed())
		Expect(copy.Copy(
			filepath.Join(
				"..",
				"..",
				"..",
				"..",
				"dist",
				"shim-credential-provider_linux_amd64_v1",
				"shim-credential-provider",
			),
			filepath.Join(providerBinDir, "shim-credential-provider"),
		)).To(Succeed())
		Expect(copy.Copy(
			filepath.Join(
				"..",
				"..",
				"..",
				"..",
				"dist",
				"static-credential-provider_linux_amd64_v1",
				"static-credential-provider",
			),
			filepath.Join(providerBinDir, "static-credential-provider"),
		)).To(Succeed())

		By("Starting KinD cluster")
		c, err := cluster.NewKinDCluster(
			ctx,
			[]kindcluster.ProviderOption{kindcluster.ProviderWithDocker()},
			[]kindcluster.CreateOption{
				kindcluster.CreateWithV1Alpha4Config(&v1alpha4.Cluster{
					KubeadmConfigPatches: []string{
						kubeadmInitPatchKubeletCredentialProviderExtraArgs,
						kubeadmJoinPatchKubeletCredentialProviderExtraArgs,
					},
					Nodes: []v1alpha4.Node{{
						Role: v1alpha4.ControlPlaneRole,
						ExtraMounts: []v1alpha4.Mount{{
							HostPath:      mirrorRegistry.CACertFile(),
							ContainerPath: "/etc/containerd/mirror-registry-ca.pem",
							Readonly:      true,
						}, {
							HostPath:      providerBinDir,
							ContainerPath: "/etc/kubernetes/image-credential-provider/",
							Readonly:      true,
						}},
					}},
					ContainerdConfigPatches: []string{
						fmt.Sprintf(
							`[plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
  endpoint = ["https://%[1]s"]
[plugins."io.containerd.grpc.v1.cri".registry.mirrors."k8s.gcr.io"]
  endpoint = ["https://%[1]s"]
[plugins."io.containerd.grpc.v1.cri".registry.mirrors."*"]
  endpoint = ["https://%[1]s"]
[plugins."io.containerd.grpc.v1.cri".registry.configs."%[1]s".tls]
  ca_file   = "/etc/containerd/mirror-registry-ca.pem"
`,
							mirrorRegistry.Address(),
						),
					},
				}),
			},
		)
		Expect(err).ShouldNot(HaveOccurred())
		DeferCleanup(func(ctx SpecContext) error {
			return c.Delete(ctx)
		}, NodeTimeout(time.Minute))
		kindCluster = c
	},
	func() {},
	NodeTimeout(time.Minute*2), GracePeriod(time.Minute*2),
)
