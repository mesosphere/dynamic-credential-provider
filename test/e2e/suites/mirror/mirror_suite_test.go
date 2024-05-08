// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package mirror_test

import (
	"context"
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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/kind/pkg/apis/config/v1alpha4"

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
	kindClusterName       string
	kindClusterRESTConfig *rest.Config
	kindClusterClient     kubernetes.Interface
	e2eConfig             e2eSetupConfig
	artifacts             goreleaser.Artifacts
	configTemplates       = template.Must(template.ParseGlob(testdataPath("*.tmpl")))
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

type staticImageCredentialsData struct {
	MirrorAddress, MirrorUsername, MirrorPassword, DockerHubPassword, DockerHubUsername string
}

type dynamicCredentialProviderConfigData struct {
	MirrorAddress string
}

type imageCredentialProviderConfigData struct {
	MirrorAddress string
}

type containerdMirrorHostsConfigData struct {
	MirrorAddress    string
	MirrorCACertPath string
}

func testdataPath(f string) string {
	return filepath.Join("testdata", f)
}

var _ = SynchronizedBeforeSuite(
	func(ctx SpecContext) []byte {
		By("Parse goreleaser artifacts")
		artifactsFileAbs, err := filepath.Abs(filepath.Join("..",
			"..",
			"..",
			"..",
			"dist", "artifacts.json"))
		Expect(err).NotTo(HaveOccurred())
		relArtifacts, err := goreleaser.ParseArtifactsFile(artifactsFileAbs)
		Expect(err).NotTo(HaveOccurred())

		artifacts = make(goreleaser.Artifacts, 0, len(relArtifacts))
		for _, a := range relArtifacts {
			if a.Path != "" {
				a.Path = filepath.Join(filepath.Dir(artifactsFileAbs), "..", a.Path)
			}
			artifacts = append(artifacts, a)
		}

		By("Starting Docker registry")
		mirrorRegistry, err := registry.NewRegistry(ctx, GinkgoT().TempDir())
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(mirrorRegistry.Delete, NodeTimeout(time.Minute))

		By("Setting up kubelet credential providers")
		providerBinDir := GinkgoT().TempDir()
		templatedFile, err := os.Create(
			filepath.Join(providerBinDir, "image-credential-provider-config.yaml"),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(configTemplates.ExecuteTemplate(
			templatedFile,
			"image-credential-provider-config.yaml.tmpl",
			imageCredentialProviderConfigData{MirrorAddress: mirrorRegistry.Address()},
		)).To(Succeed())
		templatedFile, err = os.Create(
			filepath.Join(providerBinDir, "dynamic-credential-provider-config.yaml"),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(configTemplates.ExecuteTemplate(
			templatedFile,
			"dynamic-credential-provider-config.yaml.tmpl",
			dynamicCredentialProviderConfigData{MirrorAddress: mirrorRegistry.Address()},
		)).To(Succeed())
		templatedFile, err = os.Create(
			filepath.Join(providerBinDir, "static-image-credentials.json"),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(configTemplates.ExecuteTemplate(
			templatedFile,
			"static-image-credentials.json.tmpl",
			staticImageCredentialsData{
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

		By("Setting up containerd mirror hosts configuration")
		containerdMirrorHostsConfigDir := GinkgoT().TempDir()

		containerdMirrorMirrorHostsConfigDir := filepath.Join(containerdMirrorHostsConfigDir, mirrorRegistry.Address())
		Expect(copy.Copy(
			mirrorRegistry.CACertFile(),
			filepath.Join(containerdMirrorMirrorHostsConfigDir, "ca.crt"),
		)).To(Succeed())

		containerdMirrorDefaultHostsConfigDir := filepath.Join(containerdMirrorHostsConfigDir, "_default")
		Expect(os.MkdirAll(containerdMirrorDefaultHostsConfigDir, 0o755)).To(Succeed())
		templatedFile, err = os.Create(
			filepath.Join(containerdMirrorDefaultHostsConfigDir, "hosts.toml"),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(configTemplates.ExecuteTemplate(
			templatedFile,
			"hosts.yaml.tmpl",
			containerdMirrorHostsConfigData{
				MirrorAddress:    mirrorRegistry.Address(),
				MirrorCACertPath: fmt.Sprintf("/etc/containerd/certs.d/%s/ca.crt", mirrorRegistry.Address()),
			},
		)).To(Succeed())

		By("Starting KinD cluster")
		kindCluster, kcName, kubeconfig, err := cluster.NewKinDCluster(
			ctx,
			&v1alpha4.Cluster{
				Nodes: []v1alpha4.Node{{
					Role:  v1alpha4.ControlPlaneRole,
					Image: "ghcr.io/mesosphere/kind-node:v1.30.0",
					ExtraMounts: []v1alpha4.Mount{{
						HostPath:      providerBinDir,
						ContainerPath: "/etc/kubernetes/image-credential-provider/",
					}, {
						HostPath:      containerdMirrorHostsConfigDir,
						ContainerPath: "/etc/containerd/certs.d/",
						Readonly:      true,
					}},
				}},
				KubeadmConfigPatches: []string{
					kubeadmInitPatchKubeletCredentialProviderExtraArgs,
					kubeadmJoinPatchKubeletCredentialProviderExtraArgs,
				},
				ContainerdConfigPatches: []string{
					`[plugins."io.containerd.grpc.v1.cri".registry]
  config_path = "/etc/containerd/certs.d"`,
				},
			},
		)
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(kindCluster.Delete, NodeTimeout(time.Minute))
		kindClusterName = kcName

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

func runPod(ctx context.Context, k8sClient kubernetes.Interface, image string) *corev1.Pod {
	pod, err := k8sClient.CoreV1().Pods(metav1.NamespaceDefault).
		Create(
			ctx,
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{GenerateName: "pod-"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            "container1",
						Image:           image,
						ImagePullPolicy: corev1.PullAlways,
					}},
				},
			},
			metav1.CreateOptions{},
		)
	Expect(err).NotTo(HaveOccurred())

	DeferCleanup(func(specCtx SpecContext) { //nolint:contextcheck // Idiomatic cleanup context.
		err := kindClusterClient.CoreV1().Pods(metav1.NamespaceDefault).
			Delete(specCtx, pod.GetName(), *metav1.NewDeleteOptions(0))
		Expect(err).NotTo(HaveOccurred())
	}, NodeTimeout(time.Minute))

	return pod
}

func objStatus(obj k8sruntime.Object, scheme *k8sruntime.Scheme) status.Status {
	if obj.GetObjectKind().GroupVersionKind().Group == "" {
		gvk, err := apiutil.GVKForObject(obj, scheme)
		Expect(err).NotTo(HaveOccurred())
		obj.GetObjectKind().SetGroupVersionKind(gvk)
	}

	m, err := k8sruntime.DefaultUnstructuredConverter.ToUnstructured(obj)
	Expect(err).NotTo(HaveOccurred())

	res, err := status.Compute(&unstructured.Unstructured{Object: m})
	Expect(err).NotTo(HaveOccurred())

	return res.Status
}
