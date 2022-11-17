// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package shim_test

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/mesosphere/kubelet-image-credential-provider-shim/pkg/credentialprovider/shim"
)

func ExampleGetCredentials() {
	p, err := shim.NewProviderFromConfigFile(filepath.Join("testdata", "config.yaml"))
	if err != nil {
		log.Fatal(err)
	}

	creds, err := p.GetCredentials(context.Background(), "img.v1beta1/abc/def:v1.2.3", nil)

	if err != nil {
		log.Fatal(err)
	}

	if err := json.NewEncoder(os.Stdout).Encode(creds); err != nil {
		log.Fatal(err)
	}
	// Output: {"kind":"CredentialProviderResponse","apiVersion":"credentialprovider.kubelet.k8s.io/v1beta1","cacheKeyType":"Image","cacheDuration":"5s","auth":{"img.v1beta1/abc/def:v1.2.3":{"username":"v1beta1user","password":"v1beta1password"}}}
}
