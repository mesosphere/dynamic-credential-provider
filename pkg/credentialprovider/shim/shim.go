// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package shim

import (
	"context"
	"errors"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	credentialproviderv1beta1 "k8s.io/kubelet/pkg/apis/credentialprovider/v1beta1"

	"github.com/mesosphere/kubelet-image-credential-provider-shim/apis/config/v1alpha1"
	"github.com/mesosphere/kubelet-image-credential-provider-shim/pkg/credentialprovider/plugin"
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
}

type shimProvider struct {
	cfg *v1alpha1.KubeletImageCredentialProviderShimConfig
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

	return &shimProvider{cfg: config}, nil
}

func (p shimProvider) GetCredentials(
	_ context.Context,
	img string,
	_ []string,
) (*credentialproviderv1beta1.CredentialProviderResponse, error) {
	globbedDomain, err := urlglobber.GlobbedDomainForImage(img)
	if err != nil {
		return nil, err
	}

	authMap := map[string]credentialproviderv1beta1.AuthConfig{}

	mirrorAuth, mirrorAuthFound := p.getMirrorCredentials(img)

	originAuth, originAuthFound := p.getOriginCredentials(img)

	if originAuthFound {
		authMap[img] = originAuth
	}

	if p.cfg.Mirror != nil && mirrorAuthFound {
		switch p.cfg.Mirror.MirrorCredentialsStrategy {
		case v1alpha1.MirrorCredentialsOnly:
			// Only return mirror credentials by setting the image auth for the full image name whether it is already set or
			// not.
			authMap[img] = mirrorAuth
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
			authMap[globbedDomain] = mirrorAuth
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
			authMap[img] = mirrorAuth
		default:
			return nil, fmt.Errorf(
				"%w: %q",
				ErrUnsupportedMirrorCredentialStrategy,
				p.cfg.Mirror.MirrorCredentialsStrategy,
			)
		}
	}

	return &credentialproviderv1beta1.CredentialProviderResponse{
		CacheKeyType:  credentialproviderv1beta1.ImagePluginCacheKeyType,
		CacheDuration: &metav1.Duration{Duration: 0},
		Auth:          authMap,
	}, nil
}

func (p shimProvider) getMirrorCredentials(
	img string, //nolint:unparam,revive // Placeholder for now.
) (credentialproviderv1beta1.AuthConfig, bool) { //nolint:unparam // Placeholder for now
	// If mirror is not configured then return no credentials for the mirror.
	if p.cfg.Mirror == nil {
		return credentialproviderv1beta1.AuthConfig{}, false
	}

	// TODO Call relevant credential provider plugin based on the image domain replaced with the mirror URL to get the
	// credentials for the mirror.

	return credentialproviderv1beta1.AuthConfig{}, false
}

func (p shimProvider) getOriginCredentials(
	img string, //nolint:unparam,revive // Placeholder for now
) (credentialproviderv1beta1.AuthConfig, bool) { //nolint:unparam // Placeholder for now
	// If only mirror credentials should be used then return no credentials for the origin.
	if isRegistryCredentialsOnly(p.cfg.Mirror) {
		return credentialproviderv1beta1.AuthConfig{}, false
	}

	// TODO Call relevant credential provider plugin based on the image to get the credentials for the origin.

	return credentialproviderv1beta1.AuthConfig{}, false
}

func isRegistryCredentialsOnly(cfg *v1alpha1.MirrorConfig) bool {
	return cfg != nil &&
		cfg.MirrorCredentialsStrategy == v1alpha1.MirrorCredentialsOnly
}
