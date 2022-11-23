// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package shim

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/klog/v2"
	"k8s.io/kubelet/pkg/apis/credentialprovider/install"
	credentialproviderv1beta1 "k8s.io/kubelet/pkg/apis/credentialprovider/v1beta1"

	"github.com/mesosphere/kubelet-image-credential-provider-shim/apis/config/v1alpha1"
	"github.com/mesosphere/kubelet-image-credential-provider-shim/pkg/credentialprovider/plugin"
	"github.com/mesosphere/kubelet-image-credential-provider-shim/pkg/log"
	"github.com/mesosphere/kubelet-image-credential-provider-shim/pkg/urlglobber"
)

var (
	scheme = runtime.NewScheme()
	codecs = serializer.NewCodecFactory(scheme)

	ErrUnsupportedMirrorCredentialStrategy = errors.New("unsupported mirror credential strategy")
)

//nolint:gochecknoinits // init is idiomatically used to set up schemes
func init() {
	_ = v1alpha1.AddToScheme(scheme)
	install.Install(scheme)
}

type shimProvider struct {
	cfg *v1alpha1.KubeletImageCredentialProviderShimConfig

	providersMutex sync.RWMutex
	providers      map[string]Provider
}

func NewProviderFromConfigFile(fName string) (plugin.CredentialProvider, error) {
	data, err := os.ReadFile(fName)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %q: %w", fName, err)
	}

	obj, _, err := codecs.UniversalDecoder(v1alpha1.GroupVersion).Decode(data, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config file %q: %w", fName, err)
	}

	config, ok := obj.(*v1alpha1.KubeletImageCredentialProviderShimConfig)
	if !ok {
		return nil, fmt.Errorf(
			"failed to convert %T to *KubeletImageCredentialProviderShimConfig",
			obj,
		)
	}

	shimProvider := &shimProvider{cfg: config, providers: map[string]Provider{}}
	if err := shimProvider.registerCredentialProviderPlugins(); err != nil {
		return nil, err
	}

	return shimProvider, nil
}

// registerCredentialProviderPlugins is called to register external credential provider plugins according to the
// CredentialProviderConfig config file.
func (p *shimProvider) registerCredentialProviderPlugins() error {
	if p.cfg == nil || p.cfg.CredentialProviders == nil {
		return nil
	}

	pluginBinDir := p.cfg.CredentialProviderPluginBinDir

	if _, err := os.Stat(pluginBinDir); err != nil {
		return fmt.Errorf("error inspecting plugin binary directory %q: %w", pluginBinDir, err)
	}

	for _, provider := range p.cfg.CredentialProviders.Providers {
		provider := provider // Capture range variable.

		pluginBin := filepath.Join(pluginBinDir, provider.Name)
		if _, err := os.Stat(pluginBin); err != nil {
			return fmt.Errorf("error inspecting binary executable %q: %w", pluginBin, err)
		}

		pluginProvider, err := newPluginProvider(pluginBinDir, &provider)
		if err != nil {
			return fmt.Errorf("error initializing plugin provider %q: %w", provider.Name, err)
		}

		p.registerCredentialProvider(provider.Name, pluginProvider)
	}

	return nil
}

// registerCredentialProvider registers the credential provider.
func (p *shimProvider) registerCredentialProvider(name string, provider *pluginProvider) {
	p.providersMutex.Lock()
	defer p.providersMutex.Unlock()
	_, found := p.providers[name]
	if found {
		klog.Fatalf("Credential provider %q was registered twice", name)
	}
	klog.V(log.KLogDebug).Infof("Registered credential provider %q", name)
	p.providers[name] = provider
}

func (p *shimProvider) GetCredentials(
	_ context.Context,
	img string,
	_ []string,
) (*credentialproviderv1beta1.CredentialProviderResponse, error) {
	p.providersMutex.RLock()
	defer p.providersMutex.RUnlock()

	authMap := map[string]credentialproviderv1beta1.AuthConfig{}

	mirrorAuthConfig, cacheDuration, mirrorAuthFound, err := p.getMirrorCredentialsForImage(img)
	if err != nil {
		return nil, fmt.Errorf("failed to get mirror credentials: %w", err)
	}

	var (
		originAuthConfig    credentialproviderv1beta1.AuthConfig
		originCacheDuration time.Duration
		originAuthFound     bool
	)
	if !isRegistryCredentialsOnly(p.cfg.Mirror) {
		originAuthConfig, originCacheDuration, originAuthFound, err = p.getCredentialsForImage(img)
		if err != nil {
			return nil, fmt.Errorf("failed to get origin credentials: %w", err)
		}
	}

	if originAuthFound {
		authMap[img] = originAuthConfig

		if !mirrorAuthFound || cacheDuration > originCacheDuration {
			cacheDuration = originCacheDuration
		}
	}

	if p.cfg.Mirror != nil && mirrorAuthFound {
		err := updateAuthConfigMapForMirror(
			authMap, img, p.cfg.Mirror.MirrorCredentialsStrategy, mirrorAuthConfig,
		)
		if err != nil {
			return nil, err
		}
	}

	return &credentialproviderv1beta1.CredentialProviderResponse{
		TypeMeta: metav1.TypeMeta{
			APIVersion: credentialproviderv1beta1.SchemeGroupVersion.String(),
			Kind:       "CredentialProviderResponse",
		},
		CacheKeyType:  credentialproviderv1beta1.ImagePluginCacheKeyType,
		CacheDuration: &metav1.Duration{Duration: cacheDuration},
		Auth:          authMap,
	}, nil
}

func updateAuthConfigMapForMirror(
	authMap map[string]credentialproviderv1beta1.AuthConfig,
	img string,
	mirrorCredentialsStrategy v1alpha1.MirrorCredentialsStrategy,
	mirrorAuthConfig credentialproviderv1beta1.AuthConfig,
) error {
	globbedDomain, err := urlglobber.GlobbedDomainForImage(img)
	if err != nil {
		return err
	}

	switch mirrorCredentialsStrategy {
	case v1alpha1.MirrorCredentialsOnly:
		// Only return mirror credentials by setting the image auth for the full image name whether it is already set or
		// not.
		authMap[img] = mirrorAuthConfig
	case v1alpha1.MirrorCredentialsLast:
		// Set mirror credentials for globbed domain to ensure that the mirror credentials are used last (glob matches
		// have lowest precedence).
		//
		// This means that the kubelet will first try the mirror credentials, which containerd will try against both the
		// configured mirror in containerd and the origin registry (which should fail as incorrect credentials for this
		// registry) if the image is not found in the mirror.
		//
		// If containerd fails to pull using the mirror credentials, then the kubelet will try the origin credentials,
		// which containerd will try first against the configured mirror (which should fail as incorrect credentials for
		// this registry) and then against the origin registry.
		authMap[globbedDomain] = mirrorAuthConfig
	case v1alpha1.MirrorCredentialsFirst:
		// Set mirror credentials for image to ensure that the mirror credentials are used first, and set any existing
		// origin credentials for the globbed domain to ensure they are used last (glob matches have lowest precedence).
		//
		// This means that the kubelet will first try the origin credentials, which containerd will try against both the
		// configured mirror in containerd (which should fail as incorrect credentials for this registry) and the origin
		// registry.
		//
		// If containerd fails to pull using the origin credentials, then the kubelet will try the mirror credentials,
		// which containerd will try first against the configured mirror and then against the origin registry (which
		// should fail as incorrect credentials for this registry) if the image is not found in the mirror.
		existing, found := authMap[img]
		if found {
			authMap[globbedDomain] = existing
		}
		authMap[img] = mirrorAuthConfig
	default:
		return fmt.Errorf(
			"%w: %q",
			ErrUnsupportedMirrorCredentialStrategy,
			mirrorCredentialsStrategy,
		)
	}

	return nil
}

func (p *shimProvider) getMirrorCredentialsForImage(
	img string,
) (credentialproviderv1beta1.AuthConfig, time.Duration, bool, error) {
	// If mirror is not configured then return no credentials for the mirror.
	if p.cfg.Mirror == nil {
		return credentialproviderv1beta1.AuthConfig{}, 0, false, nil
	}

	imgURL, err := urlglobber.ParsePotentiallySchemelessURL(img)
	if err != nil {
		return credentialproviderv1beta1.AuthConfig{}, 0, false, fmt.Errorf(
			"failed to parse image %q to a URL: %w",
			img,
			err,
		)
	}

	mirrorURL, err := urlglobber.ParsePotentiallySchemelessURL(p.cfg.Mirror.Endpoint)
	if err != nil {
		return credentialproviderv1beta1.AuthConfig{}, 0, false, fmt.Errorf(
			"failed to parse mirror %q to a URL: %w",
			img,
			err,
		)
	}

	imgURL.Host = mirrorURL.Host
	imgURL = mirrorURL.JoinPath(imgURL.Path)

	return p.getCredentialsForImage(strings.TrimPrefix(imgURL.String(), "//"))
}

func (p *shimProvider) getCredentialsForImage(img string) (
	authConfig credentialproviderv1beta1.AuthConfig, cacheDuration time.Duration, found bool, err error,
) {
	var longestMatchedURL string

	for _, prov := range p.providers {
		resp, err := prov.Provide(img)
		if err != nil {
			return credentialproviderv1beta1.AuthConfig{}, 0, false, fmt.Errorf(
				"failed to call plugin: %w",
				err,
			)
		}

		if resp == nil {
			continue
		}

		v1beta1Resp := &credentialproviderv1beta1.CredentialProviderResponse{}
		if err := scheme.Convert(resp, v1beta1Resp, nil); err != nil {
			return credentialproviderv1beta1.AuthConfig{},
				0,
				false,
				fmt.Errorf(
					"failed to convert response from type %T to type %T: %w",
					resp,
					v1beta1Resp,
					err,
				)
		}

		for k, v := range v1beta1Resp.Auth {
			if matched, _ := urlglobber.URLsMatchStr(k, img); !matched {
				continue
			}

			if len(longestMatchedURL) < len(k) || longestMatchedURL < k {
				longestMatchedURL = k
				authConfig = v
				cacheDuration = prov.DefaultCacheDuration()
				if resp.CacheDuration != nil {
					cacheDuration = v1beta1Resp.CacheDuration.Duration
				}
			}
		}
	}

	return authConfig, cacheDuration, len(longestMatchedURL) > 0, nil
}

func isRegistryCredentialsOnly(cfg *v1alpha1.MirrorConfig) bool {
	return cfg != nil &&
		cfg.MirrorCredentialsStrategy == v1alpha1.MirrorCredentialsOnly
}
