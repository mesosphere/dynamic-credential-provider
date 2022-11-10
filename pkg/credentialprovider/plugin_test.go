// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package credentialprovider

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubelet/pkg/apis/credentialprovider/v1alpha1"
)

type fakePlugin struct{}

func (f *fakePlugin) GetCredentials(
	ctx context.Context,
	image string,
	args []string,
) (*v1alpha1.CredentialProviderResponse, error) {
	return &v1alpha1.CredentialProviderResponse{
		CacheKeyType:  v1alpha1.RegistryPluginCacheKeyType,
		CacheDuration: &metav1.Duration{Duration: 10 * time.Minute},
		Auth: map[string]v1alpha1.AuthConfig{
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
			//nolint:lll // just a long string
			req: `{"kind":"CredentialProviderRequest","apiVersion":"credentialprovider.kubelet.k8s.io/v1alpha1","image":"test.registry.io/foobar"}`,
			//nolint:lll // just a long string
			expectedOut: `{"kind":"CredentialProviderResponse","apiVersion":"credentialprovider.kubelet.k8s.io/v1alpha1","cacheKeyType":"Registry","cacheDuration":"10m0s","auth":{"*.registry.io":{"username":"user","password":"password"}}}
`,
		},
		{
			name: "invalid kind",
			//nolint:lll // just a long string
			req:       `{"kind":"CredentialProviderFoo","apiVersion":"credentialprovider.kubelet.k8s.io/v1alpha1","image":"test.registry.io/foobar"}`,
			expectErr: ErrUnsupportedRequestKind,
		},
		{
			name: "invalid apiVersion",
			//nolint:lll // just a long string
			req:       `{"kind":"CredentialProviderRequest","apiVersion":"foo.k8s.io/v1alpha1","image":"test.registry.io/foobar"}`,
			expectErr: ErrUnsupportedAPIVersion,
		},
		{
			name: "empty image",
			//nolint:lll // just a long string
			req:       `{"kind":"CredentialProviderRequest","apiVersion":"credentialprovider.kubelet.k8s.io/v1alpha1","image":""}`,
			expectErr: ErrEmptyImageInRequest,
		},
	}

	for _, tt := range testcases {
		tt := tt // Capture range variable.

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := NewCredentialProvider(&fakePlugin{})

			out := &bytes.Buffer{}
			require.ErrorIs(
				t,
				p.runPlugin(context.TODO(), bytes.NewBufferString(tt.req), out, nil),
				tt.expectErr,
			)
			assert.Equal(t, tt.expectedOut, out.String())
		})
	}
}
