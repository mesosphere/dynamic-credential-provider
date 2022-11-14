// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package static_test

import (
	"context"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubelet/pkg/apis/credentialprovider/v1beta1"

	"github.com/mesosphere/kubelet-image-credential-provider-shim/pkg/credentialprovider/static"
)

var (
	//nolint:lll,gosec // just a long string and contains no actual credentials.
	validCredentialsStr = `{"kind":"CredentialProviderResponse","apiVersion":"credentialprovider.kubelet.k8s.io/v1beta1","cacheKeyType":"Registry","cacheDuration":"10m0s","auth":{"*.registry.io":{"username":"user","password":"password"}}}`
	validCredentials    = generateResponse("*.registry.io", 10*time.Minute, "user", "password")
)

func TestGetCredentials(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		in                string
		credentialsString string
		expectedOut       *v1beta1.CredentialProviderResponse
		expectErr         bool
	}{
		{
			name: "successful test case",
			//nolint:lll // just a long string
			in:                `{"kind":"CredentialProviderRequest","apiVersion":"credentialprovider.kubelet.k8s.io/v1beta1","image":"test.registry.io/foobar"}`,
			credentialsString: validCredentialsStr,
			expectedOut:       validCredentials,
			expectErr:         false,
		},
		{
			name: "invalid kind",
			//nolint:lll // just a long string
			in:          `{"kind":"CredentialProviderFoo","apiVersion":"credentialprovider.kubelet.k8s.io/v1beta1","image":"test.registry.io/foobar"}`,
			expectedOut: nil,
			expectErr:   true,
		},
		{
			name: "invalid apiVersion",
			//nolint:lll // just a long string
			in:          `{"kind":"CredentialProviderRequest","apiVersion":"foo.k8s.io/v1alpha1","image":"test.registry.io/foobar"}`,
			expectedOut: nil,
			expectErr:   true,
		},
		{
			name: "empty image",
			//nolint:lll // just a long string
			in:                `{"kind":"CredentialProviderRequest","apiVersion":"credentialprovider.kubelet.k8s.io/v1beta1","image":""}`,
			credentialsString: validCredentialsStr,
			expectedOut:       validCredentials,
			expectErr:         false,
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			credentialsFile := path.Join(t.TempDir(), "image-credentials.json")
			//nolint:revive // Dummy value in test file, no need for const.
			err := os.WriteFile(credentialsFile, []byte(tt.credentialsString), 0o600)
			require.NoError(t, err, "error writing temporary credentials file")

			provider, err := static.NewProvider(credentialsFile)
			require.NoError(t, err, "error initializing static credential provider")

			resp, err := provider.GetCredentials(context.Background(), "", nil)

			if err == nil && tt.expectErr {
				t.Error("expected error but got none")
			}

			assert.Equal(t, tt.expectedOut, resp)
		})
	}
}

func generateResponse(
	registry string,
	duration time.Duration,
	username string,
	password string,
) *v1beta1.CredentialProviderResponse {
	return &v1beta1.CredentialProviderResponse{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CredentialProviderResponse",
			APIVersion: "credentialprovider.kubelet.k8s.io/v1beta1",
		},
		CacheKeyType:  v1beta1.RegistryPluginCacheKeyType,
		CacheDuration: &metav1.Duration{Duration: duration},
		Auth: map[string]v1beta1.AuthConfig{
			registry: {
				Username: username,
				Password: password,
			},
		},
	}
}
