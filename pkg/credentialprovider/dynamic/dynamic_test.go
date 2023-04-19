// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package dynamic_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	credentialproviderv1 "k8s.io/kubelet/pkg/apis/credentialprovider/v1"

	"github.com/mesosphere/dynamic-credential-provider/pkg/credentialprovider/dynamic"
)

func Test_dynamicProvider_GetCredentials(t *testing.T) {
	//nolint:revive // Dummy duration ok in tests.
	expectedDummyDuration := 5 * time.Second

	const (
		dummyImageFmt                  = "img.%s/abc/def:v1.2.3"
		mirrorUser                     = "mirroruser"
		mirrorPassword                 = "mirrorpassword"
		wildcardDomain                 = "*.*"
		credentialProviderResponseKind = "CredentialProviderResponse" //nolint:gosec // No actual credentials here.
	)

	t.Parallel()

	type test struct {
		name    string
		cfgFile string
		img     string
		want    *credentialproviderv1.CredentialProviderResponse
		wantErr error
	}

	var tests []test
	for _, v := range []string{"v1", "v1beta1", "v1alpha1", "v1withpath/apath"} {
		tests = append(tests, []test{{
			name:    v + " mirror only",
			cfgFile: filepath.Join("testdata", "config-with-mirror-only.yaml"),
			img:     fmt.Sprintf(dummyImageFmt, v),
			want: &credentialproviderv1.CredentialProviderResponse{
				TypeMeta: v1.TypeMeta{
					APIVersion: credentialproviderv1.SchemeGroupVersion.String(),
					Kind:       credentialProviderResponseKind,
				},
				CacheKeyType:  credentialproviderv1.ImagePluginCacheKeyType,
				CacheDuration: &v1.Duration{Duration: expectedDummyDuration},
				Auth: map[string]credentialproviderv1.AuthConfig{
					fmt.Sprintf(dummyImageFmt, v): {Username: mirrorUser, Password: mirrorPassword},
				},
			},
		}, {
			name:    v + " mirror first",
			cfgFile: filepath.Join("testdata", "config-with-mirror-first.yaml"),
			img:     fmt.Sprintf(dummyImageFmt, v),
			want: &credentialproviderv1.CredentialProviderResponse{
				TypeMeta: v1.TypeMeta{
					APIVersion: credentialproviderv1.SchemeGroupVersion.String(),
					Kind:       credentialProviderResponseKind,
				},
				CacheKeyType:  credentialproviderv1.ImagePluginCacheKeyType,
				CacheDuration: &v1.Duration{Duration: expectedDummyDuration},
				Auth: map[string]credentialproviderv1.AuthConfig{
					fmt.Sprintf(dummyImageFmt, v): {Username: mirrorUser, Password: mirrorPassword},
					wildcardDomain:                {Username: v + "user", Password: v + "password"},
				},
			},
		}, {
			name:    v + " mirror last",
			cfgFile: filepath.Join("testdata", "config-with-mirror-last.yaml"),
			img:     fmt.Sprintf(dummyImageFmt, v),
			want: &credentialproviderv1.CredentialProviderResponse{
				TypeMeta: v1.TypeMeta{
					APIVersion: credentialproviderv1.SchemeGroupVersion.String(),
					Kind:       credentialProviderResponseKind,
				},
				CacheKeyType:  credentialproviderv1.ImagePluginCacheKeyType,
				CacheDuration: &v1.Duration{Duration: expectedDummyDuration},
				Auth: map[string]credentialproviderv1.AuthConfig{
					wildcardDomain:                {Username: mirrorUser, Password: mirrorPassword},
					fmt.Sprintf(dummyImageFmt, v): {Username: v + "user", Password: v + "password"},
				},
			},
		}, {
			name:    v + " mirror and no matching origin",
			cfgFile: filepath.Join("testdata", "config-with-mirror-last.yaml"),
			img:     "noorigin/image:v1.2.3.4",
			want: &credentialproviderv1.CredentialProviderResponse{
				TypeMeta: v1.TypeMeta{
					APIVersion: credentialproviderv1.SchemeGroupVersion.String(),
					Kind:       credentialProviderResponseKind,
				},
				CacheKeyType:  credentialproviderv1.ImagePluginCacheKeyType,
				CacheDuration: &v1.Duration{Duration: expectedDummyDuration},
				Auth: map[string]credentialproviderv1.AuthConfig{
					wildcardDomain: {Username: mirrorUser, Password: mirrorPassword},
				},
			},
		}, {
			name:    v + " no mirror",
			cfgFile: filepath.Join("testdata", "config-no-mirror.yaml"),
			img:     fmt.Sprintf(dummyImageFmt, v),
			want: &credentialproviderv1.CredentialProviderResponse{
				TypeMeta: v1.TypeMeta{
					APIVersion: credentialproviderv1.SchemeGroupVersion.String(),
					Kind:       credentialProviderResponseKind,
				},
				CacheKeyType:  credentialproviderv1.ImagePluginCacheKeyType,
				CacheDuration: &v1.Duration{Duration: expectedDummyDuration},
				Auth: map[string]credentialproviderv1.AuthConfig{
					fmt.Sprintf(dummyImageFmt, v): {Username: v + "user", Password: v + "password"},
				},
			},
		}}...,
		)
	}
	for _, tt := range tests {
		tt := tt // Capture range variable.

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p, err := dynamic.NewProviderFromConfigFile(tt.cfgFile)
			require.NoError(t, err)

			got, err := p.GetCredentials(context.Background(), tt.img, nil)
			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, got)
		})
	}
}
