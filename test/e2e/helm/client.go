// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

//nolint:revive // OK to have lots of args in test funcs.
func InstallOrUpgrade(
	ctx context.Context,
	releaseName, chartPath string,
	values map[string]interface{},
	kubeconfig, namespace string,
	log func(string, ...interface{}),
	timeout time.Duration,
) (*release.Release, error) {
	rcg := &restClientGetter{kubeconfig}
	restConfig, err := rcg.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create REST config from kubeconfig: %w", err)
	}
	kc, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client from kubeconfig: %w", err)
	}
	actionConfig := &action.Configuration{
		RESTClientGetter: rcg,
		KubeClient:       kube.New(rcg),
		Log:              log,
		Releases:         storage.Init(driver.NewSecrets(kc.CoreV1().Secrets(namespace))),
	}

	// If a release exists, upgrade it (could be no-op) or else perform an install.
	histClient := action.NewHistory(actionConfig)
	histClient.Max = 1
	_, err = histClient.Run(releaseName)
	if err != nil && !errors.Is(err, driver.ErrReleaseNotFound) {
		return nil, fmt.Errorf("failed to check if helm chart is already installed: %w", err)
	}
	upgradeRequired := err == nil

	chrt, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart from %q: %w", chartPath, err)
	}

	if upgradeRequired {
		return runUpgrade(ctx, namespace, releaseName, chrt, values, actionConfig, timeout)
	}

	return runInstall(ctx, namespace, releaseName, chrt, values, actionConfig, timeout)
}

//nolint:revive // OK to have lots of args in test funcs.
func runUpgrade(
	ctx context.Context,
	namespace string,
	releaseName string,
	chrt *chart.Chart,
	values map[string]interface{},
	cfg *action.Configuration,
	timeout time.Duration,
) (*release.Release, error) {
	upgradeAction := action.NewUpgrade(cfg)
	upgradeAction.SkipCRDs = false
	upgradeAction.Wait = true
	upgradeAction.WaitForJobs = true
	upgradeAction.Namespace = namespace
	upgradeAction.Timeout = timeout

	r, err := upgradeAction.RunWithContext(ctx, releaseName, chrt, values)
	if err != nil {
		return r, fmt.Errorf("failed to upgrade helm release: %w", err)
	}

	return r, nil
}

//nolint:revive // OK to have lots of args in test funcs.
func runInstall(
	ctx context.Context,
	namespace string,
	releaseName string,
	chrt *chart.Chart,
	values map[string]interface{},
	cfg *action.Configuration,
	timeout time.Duration,
) (*release.Release, error) {
	installAction := action.NewInstall(cfg)
	installAction.IncludeCRDs = true
	installAction.SkipCRDs = false
	installAction.Wait = true
	installAction.WaitForJobs = true
	installAction.ReleaseName = releaseName
	installAction.Namespace = namespace
	installAction.Timeout = timeout

	r, err := installAction.RunWithContext(ctx, chrt, values)
	if err != nil {
		return r, fmt.Errorf("failed to install helm release: %w", err)
	}

	return r, nil
}

type restClientGetter struct {
	kubeconfig string
}

func (rcg restClientGetter) ToRESTConfig() (*rest.Config, error) {
	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(rcg.kubeconfig))
	if err != nil {
		return nil, err
	}
	restConfig.QPS = 100   //nolint:revive // Only used here.
	restConfig.Burst = 100 //nolint:revive // Only used here.

	return restConfig, nil
}

func (rcg restClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	rc, err := rcg.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	dc, err := discovery.NewDiscoveryClientForConfig(rc)
	if err != nil {
		return nil, err
	}

	return memory.NewMemCacheClient(dc), nil
}

func (rcg restClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	rc, err := rcg.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	return apiutil.NewDiscoveryRESTMapper(rc)
}

func (rcg restClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	cc, err := clientcmd.NewClientConfigFromBytes([]byte(rcg.kubeconfig))
	if err != nil {
		panic(err)
	}

	return cc
}
