// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

/*
Copyright 2014 The Kubernetes Authors.
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

package urlglobber_test

import (
	"testing"

	"github.com/mesosphere/kubelet-image-credential-provider-shim/pkg/urlglobber"
)

const prefixKubernetesIO = "prefix.kubernetes.io"

// The content of this file is copied from
// https://github.com/kubernetes/kubernetes/blob/v1.25.4/pkg/credentialprovider/keyring_test.go#L26-L121.
// Copied rather than imported as go mod because it lives in k8s.io/kubernetes which would pull in
// lots of unnecessary dependencies.

func TestURLsMatch(t *testing.T) {
	tests := []struct {
		globURL       string
		targetURL     string
		matchExpected bool
	}{
		// match when there is no path component
		{
			globURL:       "*.kubernetes.io",
			targetURL:     prefixKubernetesIO,
			matchExpected: true,
		},
		{
			globURL:       "prefix.*.io",
			targetURL:     prefixKubernetesIO,
			matchExpected: true,
		},
		{
			globURL:       "prefix.kubernetes.*",
			targetURL:     prefixKubernetesIO,
			matchExpected: true,
		},
		{
			globURL:       "*-good.kubernetes.io",
			targetURL:     "prefix-good.kubernetes.io",
			matchExpected: true,
		},
		// match with path components
		{
			globURL:       "*.kubernetes.io/blah",
			targetURL:     prefixKubernetesIO + "/blah",
			matchExpected: true,
		},
		{
			globURL:       "prefix.*.io/foo",
			targetURL:     prefixKubernetesIO + "/foo/bar",
			matchExpected: true,
		},
		// match with path components and ports
		{
			globURL:       "*.kubernetes.io:1111/blah",
			targetURL:     prefixKubernetesIO + ":1111/blah",
			matchExpected: true,
		},
		{
			globURL:       "prefix.*.io:1111/foo",
			targetURL:     prefixKubernetesIO + ":1111/foo/bar",
			matchExpected: true,
		},
		// no match when number of parts mismatch
		{
			globURL:       "*.kubernetes.io",
			targetURL:     "kubernetes.io",
			matchExpected: false,
		},
		{
			globURL:       "*.*.kubernetes.io",
			targetURL:     prefixKubernetesIO,
			matchExpected: false,
		},
		{
			globURL:       "*.*.kubernetes.io",
			targetURL:     "kubernetes.io",
			matchExpected: false,
		},
		// no match when some parts mismatch
		{
			globURL:       "kubernetes.io",
			targetURL:     "kubernetes.com",
			matchExpected: false,
		},
		{
			globURL:       "k*.io",
			targetURL:     "quay.io",
			matchExpected: false,
		},
		// no match when ports mismatch
		{
			globURL:       "*.kubernetes.io:1234/blah",
			targetURL:     prefixKubernetesIO + ":1111/blah",
			matchExpected: false,
		},
		{
			globURL:       "prefix.*.io/foo",
			targetURL:     prefixKubernetesIO + ":1111/foo/bar",
			matchExpected: false,
		},
		// match when there is a scheme specified.
		{
			globURL:       "https://*.kubernetes.io",
			targetURL:     prefixKubernetesIO,
			matchExpected: true,
		},
	}
	for _, test := range tests {
		matched, _ := urlglobber.URLsMatchStr(test.globURL, test.targetURL)
		if matched != test.matchExpected {
			t.Errorf("Expected match result of %s and %s to be %t, but was %t",
				test.globURL, test.targetURL, test.matchExpected, matched)
		}
	}
}
