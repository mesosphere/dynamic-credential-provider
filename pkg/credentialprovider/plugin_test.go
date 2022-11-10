// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package credentialprovider

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
	testcases := []struct {
		name        string
		in          *bytes.Buffer
		expectedOut []byte
		expectErr   bool
	}{
		{
			name: "successful test case",
			//nolint:lll // just a long string
			in: bytes.NewBufferString(
				`{"kind":"CredentialProviderRequest","apiVersion":"credentialprovider.kubelet.k8s.io/v1alpha1","image":"test.registry.io/foobar"}`,
			),
			//nolint:lll // just a long string
			expectedOut: []byte(
				`{"kind":"CredentialProviderResponse","apiVersion":"credentialprovider.kubelet.k8s.io/v1alpha1","cacheKeyType":"Registry","cacheDuration":"10m0s","auth":{"*.registry.io":{"username":"user","password":"password"}}}
`,
			),
			expectErr: false,
		},
		{
			name: "invalid kind",
			//nolint:lll // just a long string
			in: bytes.NewBufferString(
				`{"kind":"CredentialProviderFoo","apiVersion":"credentialprovider.kubelet.k8s.io/v1alpha1","image":"test.registry.io/foobar"}`,
			),
			expectedOut: nil,
			expectErr:   true,
		},
		{
			name: "invalid apiVersion",
			in: bytes.NewBufferString(
				`{"kind":"CredentialProviderRequest","apiVersion":"foo.k8s.io/v1alpha1","image":"test.registry.io/foobar"}`,
			),
			expectedOut: nil,
			expectErr:   true,
		},
		{
			name: "empty image",
			in: bytes.NewBufferString(
				`{"kind":"CredentialProviderRequest","apiVersion":"credentialprovider.kubelet.k8s.io/v1alpha1","image":""}`,
			),
			expectedOut: nil,
			expectErr:   true,
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			p := NewCredentialProvider(&fakePlugin{})

			out := &bytes.Buffer{}
			err := p.runPlugin(context.TODO(), tt.in, out, nil)
			if err != nil && !tt.expectErr {
				t.Fatal(err)
			}

			if err == nil && tt.expectErr {
				t.Error("expected error but got none")
			}

			assert.Equal(t, string(tt.expectedOut), out.String())
		})
	}
}
