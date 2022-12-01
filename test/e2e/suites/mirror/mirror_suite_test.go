// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package mirror_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"text/template"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/otiai10/copy"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
	kindcluster "sigs.k8s.io/kind/pkg/cluster"

	"github.com/mesosphere/dynamic-credential-provider/test/e2e/cluster"
	"github.com/mesosphere/dynamic-credential-provider/test/e2e/env"
	"github.com/mesosphere/dynamic-credential-provider/test/e2e/goreleaser"
	"github.com/mesosphere/dynamic-credential-provider/test/e2e/registry"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mirror Suite")
}

type e2eSetupConfig struct {
	Registry   e2eRegistryConfig `json:"registry"`
	Kubeconfig string            `json:"kubeconfig"`
}

type e2eRegistryConfig struct {
	Username        string `json:"username"`
	Password        string `json:"password"`
	Address         string `json:"address"`
	HostPortAddress string `json:"hostPortAddress"`
	CACertFile      string `json:"caCertFile"`
}

var (
	kindClusterRESTConfig *rest.Config
	kindClusterClient     kubernetes.Interface
	e2eConfig             e2eSetupConfig
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
	func(ctx SpecContext) []byte {
		By("Parse goreleaser artifacts")
		artifacts, err := goreleaser.ParseArtifactsFile(filepath.Join("..",
			"..",
			"..",
			"..",
			"dist",
			"artifacts.json",
		))
		Expect(err).NotTo(HaveOccurred())

		By("Starting Docker registry")
		mirrorRegistry := registry.NewRegistry(ctx)

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
			struct{ MirrorAddress string }{MirrorAddress: mirrorRegistry.Address()},
		)).To(Succeed())
		templatedFile, err = os.Create(
			filepath.Join(providerBinDir, "dynamic-credential-provider-config.yaml"),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(configTmpl.ExecuteTemplate(
			templatedFile,
			"dynamic-credential-provider-config.yaml.tmpl",
			struct{ MirrorAddress string }{MirrorAddress: mirrorRegistry.Address()},
		)).To(Succeed())
		templatedFile, err = os.Create(
			filepath.Join(providerBinDir, "static-image-credentials.json"),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(configTmpl.ExecuteTemplate(
			templatedFile,
			"static-image-credentials.json.tmpl",
			struct {
				MirrorAddress, MirrorUsername, MirrorPassword, DockerHubPassword, DockerHubUsername string
			}{
				MirrorAddress:     mirrorRegistry.Address(),
				MirrorUsername:    mirrorRegistry.Username(),
				MirrorPassword:    mirrorRegistry.Password(),
				DockerHubUsername: env.DockerHubUsername(),
				DockerHubPassword: env.DockerHubPassword(),
			},
		)).To(Succeed())
		Expect(templatedFile.Close()).To(Succeed())
		bin, ok := artifacts.SelectBinary("dynamic-credential-provider", "linux", runtime.GOARCH)
		Expect(ok).To(BeTrue())
		Expect(copy.Copy(
			bin.Path,
			filepath.Join(providerBinDir, "dynamic-credential-provider"),
		)).To(Succeed())

		bin, ok = artifacts.SelectBinary("static-credential-provider", "linux", runtime.GOARCH)
		Expect(ok).To(BeTrue())
		Expect(copy.Copy(
			bin.Path,
			filepath.Join(providerBinDir, "static-credential-provider"),
		)).To(Succeed())

		By("Starting KinD cluster")
		_, kubeconfig := cluster.NewKinDCluster(
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

		configBytes, _ := json.Marshal(e2eSetupConfig{
			Registry: e2eRegistryConfig{
				Username:        mirrorRegistry.Username(),
				Password:        mirrorRegistry.Password(),
				Address:         mirrorRegistry.Address(),
				HostPortAddress: mirrorRegistry.HostPortAddress(),
				CACertFile:      mirrorRegistry.CACertFile(),
			},
			Kubeconfig: kubeconfig,
		})

		return configBytes
	},
	func(configBytes []byte) {
		Expect(json.Unmarshal(configBytes, &e2eConfig)).To(Succeed())

		var err error
		kindClusterRESTConfig, err = clientcmd.RESTConfigFromKubeConfig(
			[]byte(e2eConfig.Kubeconfig),
		)
		Expect(err).NotTo(HaveOccurred())
		kindClusterClient, err = kubernetes.NewForConfig(kindClusterRESTConfig)
		Expect(err).NotTo(HaveOccurred())
	},
	NodeTimeout(time.Minute*2), GracePeriod(time.Minute*2),
)
