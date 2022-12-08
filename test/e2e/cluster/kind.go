// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/pkg/namesgenerator"
	"go.uber.org/multierr"
	yaml "gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	kindv1alpha4 "sigs.k8s.io/kind/pkg/apis/config/v1alpha4"

	"github.com/mesosphere/dynamic-credential-provider/test/e2e/seedrng"
)

// kindCluster represents a KinD cluster.
type kindCluster struct {
	name string
}

// NewKinDCluster creates a KinD cluster and returns a Cluster ready to be used.
func NewKinDCluster(
	ctx context.Context,
	cfg *kindv1alpha4.Cluster,
) (kc Cluster, name, kubeconfig string, err error) {
	seedrng.EnsureSeeded()

	if cfg.Name == "" {
		cfg.Name = strings.ReplaceAll(namesgenerator.GetRandomName(0), "_", "-")
	}

	// Do not export kubeconfig to file by default, makes cleanup easier.
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("%s-kindcluster-*", cfg.Name))
	if err != nil {
		return nil, "", "", fmt.Errorf(
			"failed to create temporary directory for KinD cluster configuration: %w",
			err,
		)
	}
	defer os.RemoveAll(tempDir)

	defaultKinDClusterConfig(cfg)
	cfgFile := filepath.Join(tempDir, "cluster.yaml")
	cfgBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, "", "", fmt.Errorf(
			"failed to marshal KinD cluster configuration to YAML: %w",
			err,
		)
	}
	if err := os.WriteFile(cfgFile, cfgBytes, 0o400); err != nil { //nolint:revive // 0400 is standard read-only perms.
		return nil, "", "", fmt.Errorf(
			"failed to write KinD cluster configuration to file: %w",
			err,
		)
	}

	tempKubeconfig := filepath.Join(tempDir, "kubeconfig")

	cmd := exec.CommandContext( //nolint:gosec // Only used in tests so safe.
		ctx,
		"kind", "create", "cluster",
		"--name", cfg.Name,
		"--kubeconfig", tempKubeconfig,
		"--config", cfgFile,
		"--retain",
	)

	err = cmd.Run()
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to create KinD cluster: %w", err)
	}

	kubeconfigBytes, err := os.ReadFile(tempKubeconfig)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to get KinD cluster kubeconfig: %w", err)
	}
	kubeconfig = string(kubeconfigBytes)

	kc = &kindCluster{
		name: cfg.Name,
	}

	if err := waitForDefaultServiceAccountToExist(ctx, kubeconfig); err != nil {
		if deleteErr := kc.Delete( //nolint:contextcheck // Best effort background deletion.
			context.Background(),
		); deleteErr != nil {
			err = multierr.Combine(err, deleteErr)
		}
		return nil, "", "", err
	}

	return kc, cfg.Name, kubeconfig, nil
}

func (c *kindCluster) Delete(ctx context.Context) error {
	tempKubeconfig, err := os.CreateTemp("", fmt.Sprintf("%s-kindcluster-kubeconfig-*", c.name))
	if err != nil {
		return fmt.Errorf("failed to create temporary file for KinD cluster kubeconfig: %w", err)
	}
	if err := tempKubeconfig.Close(); err != nil {
		return fmt.Errorf("failed to create temporary file for KinD cluster kubeconfig: %w", err)
	}
	defer os.Remove(tempKubeconfig.Name())

	cmd := exec.CommandContext( //nolint:gosec // Only used in tests so safe.
		ctx,
		"kind",
		"delete", "cluster",
		"--name", c.name,
		"--kubeconfig", tempKubeconfig.Name(),
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete KinD cluster: %w", err)
	}

	return nil
}

func waitForDefaultServiceAccountToExist(ctx context.Context, kubeconfig string) error {
	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return fmt.Errorf("failed to create REST config from kubeconfig: %w", err)
	}

	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client from kubeconfig: %w", err)
	}

	return wait.PollInfiniteWithContext(
		ctx,
		time.Second*1,
		func(ctx context.Context) (bool, error) {
			_, getSAErr := kubeClient.CoreV1().ServiceAccounts(metav1.NamespaceDefault).
				Get(ctx, "default", metav1.GetOptions{})
			if getSAErr == nil {
				return true, nil
			}
			if errors.IsNotFound(getSAErr) {
				return false, nil
			}
			return false, getSAErr
		},
	)
}

func defaultKinDClusterConfig(cfg *kindv1alpha4.Cluster) {
	kindv1alpha4.SetDefaultsCluster(cfg)

	if cfg.APIVersion == "" {
		cfg.APIVersion = "kind.x-k8s.io/v1alpha4"
	}
	if cfg.Kind == "" {
		cfg.Kind = "Cluster"
	}
}
