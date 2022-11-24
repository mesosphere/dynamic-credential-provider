// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package shim_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	credentialproviderv1beta1 "k8s.io/kubelet/pkg/apis/credentialprovider/v1beta1"

	"github.com/mesosphere/kubelet-image-credential-provider-shim/pkg/credentialprovider/shim"
)

func Test_shimProvider_GetCredentials(t *testing.T) {
	//nolint:revive // Dummy duration ok in tests.
	expectedDummyDuration := 5 * time.Second

	const (
		dummyImage                     = "img.v1beta1/abc/def:v1.2.3"
		mirrorUser                     = "mirroruser"
		mirrorPassword                 = "mirrorpassword"
		wildcardDomain                 = "*.*"
		credentialProviderResponseKind = "CredentialProviderResponse" //nolint:gosec // No actual credentials here.
	)

	t.Parallel()
	tests := []struct {
		name    string
		cfgFile string
		img     string
		want    *credentialproviderv1beta1.CredentialProviderResponse
		wantErr error
	}{{
		name:    "mirror only",
		cfgFile: filepath.Join("testdata", "config-with-mirror-only.yaml"),
		img:     dummyImage,
		want: &credentialproviderv1beta1.CredentialProviderResponse{
			TypeMeta: v1.TypeMeta{
				APIVersion: credentialproviderv1beta1.SchemeGroupVersion.String(),
				Kind:       credentialProviderResponseKind,
			},
			CacheKeyType:  credentialproviderv1beta1.ImagePluginCacheKeyType,
			CacheDuration: &v1.Duration{Duration: expectedDummyDuration},
			Auth: map[string]credentialproviderv1beta1.AuthConfig{
				dummyImage: {Username: mirrorUser, Password: mirrorPassword},
			},
		},
	}, {
		name:    "mirror first",
		cfgFile: filepath.Join("testdata", "config-with-mirror-first.yaml"),
		img:     dummyImage,
		want: &credentialproviderv1beta1.CredentialProviderResponse{
			TypeMeta: v1.TypeMeta{
				APIVersion: credentialproviderv1beta1.SchemeGroupVersion.String(),
				Kind:       credentialProviderResponseKind,
			},
			CacheKeyType:  credentialproviderv1beta1.ImagePluginCacheKeyType,
			CacheDuration: &v1.Duration{Duration: expectedDummyDuration},
			Auth: map[string]credentialproviderv1beta1.AuthConfig{
				dummyImage:     {Username: mirrorUser, Password: mirrorPassword},
				wildcardDomain: {Username: "v1beta1user", Password: "v1beta1password"},
			},
		},
	}, {
		name:    "mirror last",
		cfgFile: filepath.Join("testdata", "config-with-mirror-last.yaml"),
		img:     dummyImage,
		want: &credentialproviderv1beta1.CredentialProviderResponse{
			TypeMeta: v1.TypeMeta{
				APIVersion: credentialproviderv1beta1.SchemeGroupVersion.String(),
				Kind:       credentialProviderResponseKind,
			},
			CacheKeyType:  credentialproviderv1beta1.ImagePluginCacheKeyType,
			CacheDuration: &v1.Duration{Duration: expectedDummyDuration},
			Auth: map[string]credentialproviderv1beta1.AuthConfig{
				wildcardDomain: {Username: mirrorUser, Password: mirrorPassword},
				dummyImage:     {Username: "v1beta1user", Password: "v1beta1password"},
			},
		},
	}, {
		name:    "mirror and no matching origin",
		cfgFile: filepath.Join("testdata", "config-with-mirror-last.yaml"),
		img:     "noorigin/image:v1.2.3.4",
		want: &credentialproviderv1beta1.CredentialProviderResponse{
			TypeMeta: v1.TypeMeta{
				APIVersion: credentialproviderv1beta1.SchemeGroupVersion.String(),
				Kind:       credentialProviderResponseKind,
			},
			CacheKeyType:  credentialproviderv1beta1.ImagePluginCacheKeyType,
			CacheDuration: &v1.Duration{Duration: expectedDummyDuration},
			Auth: map[string]credentialproviderv1beta1.AuthConfig{
				wildcardDomain: {Username: mirrorUser, Password: mirrorPassword},
			},
		},
	}, {
		name:    "no mirror",
		cfgFile: filepath.Join("testdata", "config-no-mirror.yaml"),
		img:     dummyImage,
		want: &credentialproviderv1beta1.CredentialProviderResponse{
			TypeMeta: v1.TypeMeta{
				APIVersion: credentialproviderv1beta1.SchemeGroupVersion.String(),
				Kind:       credentialProviderResponseKind,
			},
			CacheKeyType:  credentialproviderv1beta1.ImagePluginCacheKeyType,
			CacheDuration: &v1.Duration{Duration: expectedDummyDuration},
			Auth: map[string]credentialproviderv1beta1.AuthConfig{
				dummyImage: {Username: "v1beta1user", Password: "v1beta1password"},
			},
		},
	}}
	for _, tt := range tests {
		tt := tt // Capture range variable.

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p, err := shim.NewProviderFromConfigFile(tt.cfgFile)
			require.NoError(t, err)

			got, err := p.GetCredentials(context.Background(), tt.img, nil)
			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, got)
		})
	}
}
