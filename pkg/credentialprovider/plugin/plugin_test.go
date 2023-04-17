// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/kubelet/pkg/apis/credentialprovider/v1"
)

type fakePlugin struct{}

func (fakePlugin) GetCredentials(
	_ context.Context,
	_ string,
	_ []string,
) (*v1.CredentialProviderResponse, error) {
	return &v1.CredentialProviderResponse{
		CacheKeyType: v1.RegistryPluginCacheKeyType,
		//nolint:revive // Dummy value in test file, no need for const.
		CacheDuration: &metav1.Duration{Duration: 10 * time.Minute},
		Auth: map[string]v1.AuthConfig{
			"*.registry.io": {
				Username: "user",
				Password: "password",
			},
		},
	}, nil
}

func Test_runPlugin(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name        string
		req         string
		expectedOut string
		expectErr   error
	}{
		{
			name: "successful test case",
			//nolint:lll // Just a long string.
			req: `{"kind":"CredentialProviderRequest","apiVersion":"credentialprovider.kubelet.k8s.io/v1","image":"test.registry.io/foobar"}`,
			//nolint:lll // Just a long string.
			expectedOut: `{"kind":"CredentialProviderResponse","apiVersion":"credentialprovider.kubelet.k8s.io/v1","cacheKeyType":"Registry","cacheDuration":"10m0s","auth":{"*.registry.io":{"username":"user","password":"password"}}}
`,
		},
		{
			name: "invalid kind",
			//nolint:lll // Just a long string.
			req:       `{"kind":"CredentialProviderFoo","apiVersion":"credentialprovider.kubelet.k8s.io/v1","image":"test.registry.io/foobar"}`,
			expectErr: ErrUnsupportedRequestKind,
		},
		{
			name: "invalid apiVersion",
			//nolint:lll // Just a long string.
			req:       `{"kind":"CredentialProviderRequest","apiVersion":"foo.k8s.io/v1alpha1","image":"test.registry.io/foobar"}`,
			expectErr: ErrUnsupportedAPIVersion,
		},
		{
			name:      "empty image",
			req:       `{"kind":"CredentialProviderRequest","apiVersion":"credentialprovider.kubelet.k8s.io/v1","image":""}`,
			expectErr: ErrEmptyImageInRequest,
		},
	}

	for _, tt := range testcases {
		tt := tt // Capture range variable.

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := NewProvider(fakePlugin{})

			out := &bytes.Buffer{}
			require.ErrorIs(
				t,
				p.runPlugin(context.Background(), bytes.NewBufferString(tt.req), out, nil),
				tt.expectErr,
			)
			assert.Equal(t, tt.expectedOut, out.String())
		})
	}
}
