// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/pkg/namesgenerator"
	g "github.com/onsi/ginkgo/v2"
	gm "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/kind/pkg/cluster"

	"github.com/mesosphere/dynamic-credential-provider/test/e2e/seedrng"
)

// kindCluster represents a KinD cluster.
type kindCluster struct {
	name     string
	provider *cluster.Provider
}

// NewKinDCluster creates a KinD cluster and returns a Cluster ready to be used.
func NewKinDCluster(
	ctx context.Context,
	providerOpts []cluster.ProviderOption,
	createOpts []cluster.CreateOption,
) (kc Cluster, kubeconfig string) {
	seedrng.EnsureSeeded()

	name := strings.ReplaceAll(namesgenerator.GetRandomName(0), "_", "-")

	provider := cluster.NewProvider(providerOpts...)

	// Do not export kubeconfig to file by default, makes cleanup easier. This can be overridden by using create option
	// to configure this via `cluster.CreateWithKubeconfigPath` when calling `NewKindCluster`.
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("%s-kubeconfig-*", name))
	gm.Expect(err).NotTo(gm.HaveOccurred())
	g.DeferCleanup(os.RemoveAll, tempDir)
	tempKubeconfig := filepath.Join(tempDir, "kubeconfig")
	mergedCreateOpts := []cluster.CreateOption{cluster.CreateWithKubeconfigPath(tempKubeconfig)}
	mergedCreateOpts = append(mergedCreateOpts, createOpts...)

	g.DeferCleanup(provider.Delete, name, "")
	kindClusterErr := make(chan error, 1)
	go func() {
		kindClusterErr <- provider.Create(name, mergedCreateOpts...)
	}()

	select {
	case <-ctx.Done():
		return nil, ""
	case err := <-kindClusterErr:
		gm.Expect(err).NotTo(gm.HaveOccurred())
	}

	kubeconfig, err = provider.KubeConfig(name, false)
	gm.Expect(err).NotTo(gm.HaveOccurred())

	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	gm.Expect(err).NotTo(gm.HaveOccurred())

	kubeClient, err := kubernetes.NewForConfig(restConfig)
	gm.Expect(err).NotTo(gm.HaveOccurred())

	err = wait.PollInfiniteWithContext(ctx, time.Second*1, func(ctx context.Context) (bool, error) {
		_, getSAErr := kubeClient.CoreV1().ServiceAccounts(metav1.NamespaceDefault).
			Get(ctx, "default", metav1.GetOptions{})
		if getSAErr == nil {
			return true, nil
		}
		if errors.IsNotFound(getSAErr) {
			return false, nil
		}
		return false, getSAErr
	})
	gm.Expect(err).NotTo(gm.HaveOccurred())

	return &kindCluster{
		name:     name,
		provider: provider,
	}, kubeconfig
}

func (c *kindCluster) Delete(context.Context) error {
	return c.provider.Delete(c.name, "")
}
