// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package urlglobber_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mesosphere/dynamic-credential-provider/pkg/urlglobber"
)

func TestGlobbedDomainForImage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		img         string
		wantGlobbed string
		wantErr     error
	}{{
		name:    "Empty URL",
		wantErr: urlglobber.ErrInvalidImageReference,
	}, {
		name:        "Simple image",
		img:         prefixKubernetesIO + "/foo/bar:v1.2.3",
		wantGlobbed: "*.*.*",
	}, {
		name:        "Simple image with port",
		img:         prefixKubernetesIO + ":1111/foo/bar:v1.2.3",
		wantGlobbed: "*.*.*:1111",
	}, {
		name:        "Image from docker hub with no domain",
		img:         "foo/bar:v1.2.3",
		wantGlobbed: "*.*", // To match docker.io.
	}, {
		name:        "Image from docker hub with no domain or path",
		img:         "bar:v1.2.3",
		wantGlobbed: "*.*", // To match docker.io.
	}}
	for _, tt := range tests {
		tt := tt // Capture range variable.
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			globbed, err := urlglobber.GlobbedDomainForImage(tt.img)
			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantGlobbed, globbed)
		})
	}
}
