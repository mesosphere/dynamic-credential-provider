// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

/*
Copyright 2020 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package webhooks

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	kubeletconfigv1 "k8s.io/kubelet/config/v1"
	"k8s.io/kubelet/pkg/apis/credentialprovider/v1alpha1"
)

const (
	dummyRegistryDomain = "foobar.registry.io"
	dummyName           = "foobar"
)

// Copied from
// https://github.com/kubernetes/kubernetes/blob/v1.25.4/pkg/credentialprovider/plugin/config_test.go#L303-L481.
func Test_validateCredentialProviderConfig(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name      string
		config    *kubeletconfigv1.CredentialProviderConfig
		shouldErr bool
	}{
		{
			name:      "no providers provided",
			config:    &kubeletconfigv1.CredentialProviderConfig{},
			shouldErr: true,
		},
		{
			name: "no matchImages provided",
			config: &kubeletconfigv1.CredentialProviderConfig{
				Providers: []kubeletconfigv1.CredentialProvider{
					{
						Name:                 dummyName,
						MatchImages:          []string{},
						DefaultCacheDuration: &metav1.Duration{Duration: time.Minute},
						APIVersion:           v1alpha1.SchemeGroupVersion.String(),
					},
				},
			},
			shouldErr: true,
		},
		{
			name: "no default cache duration provided",
			config: &kubeletconfigv1.CredentialProviderConfig{
				Providers: []kubeletconfigv1.CredentialProvider{
					{
						Name:        dummyName,
						MatchImages: []string{dummyRegistryDomain},
						APIVersion:  v1alpha1.SchemeGroupVersion.String(),
					},
				},
			},
			shouldErr: true,
		},
		{
			name: "name contains '/'",
			config: &kubeletconfigv1.CredentialProviderConfig{
				Providers: []kubeletconfigv1.CredentialProvider{
					{
						Name:                 "foo/../bar",
						MatchImages:          []string{dummyRegistryDomain},
						DefaultCacheDuration: &metav1.Duration{Duration: time.Minute},
						APIVersion:           v1alpha1.SchemeGroupVersion.String(),
					},
				},
			},
			shouldErr: true,
		},
		{
			name: "name is '.'",
			config: &kubeletconfigv1.CredentialProviderConfig{
				Providers: []kubeletconfigv1.CredentialProvider{
					{
						Name:                 ".",
						MatchImages:          []string{dummyRegistryDomain},
						DefaultCacheDuration: &metav1.Duration{Duration: time.Minute},
						APIVersion:           v1alpha1.SchemeGroupVersion.String(),
					},
				},
			},
			shouldErr: true,
		},
		{
			name: "name is '..'",
			config: &kubeletconfigv1.CredentialProviderConfig{
				Providers: []kubeletconfigv1.CredentialProvider{
					{
						Name:                 "..",
						MatchImages:          []string{dummyRegistryDomain},
						DefaultCacheDuration: &metav1.Duration{Duration: time.Minute},
						APIVersion:           v1alpha1.SchemeGroupVersion.String(),
					},
				},
			},
			shouldErr: true,
		},
		{
			name: "name contains spaces",
			config: &kubeletconfigv1.CredentialProviderConfig{
				Providers: []kubeletconfigv1.CredentialProvider{
					{
						Name:                 "foo bar",
						MatchImages:          []string{dummyRegistryDomain},
						DefaultCacheDuration: &metav1.Duration{Duration: time.Minute},
						APIVersion:           v1alpha1.SchemeGroupVersion.String(),
					},
				},
			},
			shouldErr: true,
		},
		{
			name: "no apiVersion",
			config: &kubeletconfigv1.CredentialProviderConfig{
				Providers: []kubeletconfigv1.CredentialProvider{
					{
						Name:                 dummyName,
						MatchImages:          []string{dummyRegistryDomain},
						DefaultCacheDuration: &metav1.Duration{Duration: time.Minute},
						APIVersion:           "",
					},
				},
			},
			shouldErr: true,
		},
		{
			name: "invalid apiVersion",
			config: &kubeletconfigv1.CredentialProviderConfig{
				Providers: []kubeletconfigv1.CredentialProvider{
					{
						Name:                 dummyName,
						MatchImages:          []string{dummyRegistryDomain},
						DefaultCacheDuration: &metav1.Duration{Duration: time.Minute},
						APIVersion:           "credentialprovider.kubelet.k8s.io/v1alpha0",
					},
				},
			},
			shouldErr: true,
		},
		{
			name: "negative default cache duration",
			config: &kubeletconfigv1.CredentialProviderConfig{
				Providers: []kubeletconfigv1.CredentialProvider{
					{
						Name:                 dummyName,
						MatchImages:          []string{dummyRegistryDomain},
						DefaultCacheDuration: &metav1.Duration{Duration: -1 * time.Minute},
						APIVersion:           v1alpha1.SchemeGroupVersion.String(),
					},
				},
			},
			shouldErr: true,
		},
		{
			name: "invalid match image",
			config: &kubeletconfigv1.CredentialProviderConfig{
				Providers: []kubeletconfigv1.CredentialProvider{
					{
						Name:                 dummyName,
						MatchImages:          []string{"%invalid%"},
						DefaultCacheDuration: &metav1.Duration{Duration: time.Minute},
						APIVersion:           v1alpha1.SchemeGroupVersion.String(),
					},
				},
			},
			shouldErr: true,
		},
		{
			name: "valid config",
			config: &kubeletconfigv1.CredentialProviderConfig{
				Providers: []kubeletconfigv1.CredentialProvider{
					{
						Name:                 dummyName,
						MatchImages:          []string{dummyRegistryDomain},
						DefaultCacheDuration: &metav1.Duration{Duration: time.Minute},
						APIVersion:           v1alpha1.SchemeGroupVersion.String(),
					},
				},
			},
			shouldErr: false,
		},
	}

	for _, tt := range testcases {
		tt := tt // Capture range variable.

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			errs := validateCredentialProviderConfig(context.Background(), tt.config, &field.Path{})

			if tt.shouldErr && len(errs) == 0 {
				t.Errorf("expected error but got none")
			} else if !tt.shouldErr && len(errs) > 0 {
				t.Errorf("expected no error but received errors: %v", errs.ToAggregate())
			}
		})
	}
}
