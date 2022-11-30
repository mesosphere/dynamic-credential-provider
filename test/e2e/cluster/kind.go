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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/kind/pkg/cluster"

	"github.com/mesosphere/kubelet-image-credential-provider-shim/test/e2e/seedrng"
)

// kindCluster represents a KinD cluster.
type kindCluster struct {
	name     string
	provider *cluster.Provider
}

// NewKinDCluster creates a KinD cluster and returns a Cluster ready to be used.
//
//nolint:revive // Complex function is ok in this test file.
func NewKinDCluster(
	ctx context.Context,
	providerOpts []cluster.ProviderOption,
	createOpts []cluster.CreateOption,
) (Cluster, string, error) {
	seedrng.EnsureSeeded()

	name := strings.ReplaceAll(namesgenerator.GetRandomName(0), "_", "-")

	provider := cluster.NewProvider(providerOpts...)

	// Do not export kubeconfig to file by default, makes cleanup easier. This can be overridden by using create option
	// to configure this via `cluster.CreateWithKubeconfigPath` when calling `NewKindCluster`.
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("%s-kubeconfig-*", name))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)
	tempKubeconfig := filepath.Join(tempDir, "kubeconfig")
	mergedCreateOpts := []cluster.CreateOption{cluster.CreateWithKubeconfigPath(tempKubeconfig)}
	mergedCreateOpts = append(mergedCreateOpts, createOpts...)

	kindClusterErr := make(chan error, 1)
	go func() {
		kindClusterErr <- provider.Create(name, mergedCreateOpts...)
	}()

	select {
	case <-ctx.Done():
		if err := provider.Delete(name, ""); err != nil {
			return nil, "", fmt.Errorf("failed to delete KinD cluster after spec timeout: %w", err)
		}
		return nil, "", nil
	case err := <-kindClusterErr:
		if err != nil {
			return nil, "", fmt.Errorf("failed to create KinD cluster: %w", err)
		}
	}

	const warningDeleteKinD = "WARNING: failed to delete KinD cluster: %v"
	kubeconfig, err := provider.KubeConfig(name, false)
	if err != nil {
		if deleteErr := provider.Delete(name, ""); deleteErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, warningDeleteKinD, deleteErr)
		}
		return nil, "", fmt.Errorf("failed to retrieve kubeconfig: %w", err)
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		if deleteErr := provider.Delete(name, ""); deleteErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, warningDeleteKinD, deleteErr)
		}
		return nil, "", fmt.Errorf("failed to build REST config from kubeconfig: %w", err)
	}

	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		if deleteErr := provider.Delete(name, ""); deleteErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, warningDeleteKinD, deleteErr)
		}
		return nil, "", fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

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
	if err != nil {
		if err := provider.Delete(name, ""); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, warningDeleteKinD, err)
		}
		return nil, "", fmt.Errorf("failed to wait for default service account to exist: %w", err)
	}

	return &kindCluster{
		name:     name,
		provider: provider,
	}, kubeconfig, nil
}

func (c *kindCluster) Delete(context.Context) error {
	return c.provider.Delete(c.name, "")
}
