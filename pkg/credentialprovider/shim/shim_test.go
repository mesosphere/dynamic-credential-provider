// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package shim

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

func ExampleCredentialProvider_GetCredentials() {
	p, err := NewProviderFromConfigFile(filepath.Join("testdata", "config.yaml"))
	if err != nil {
		log.Fatal(err) //nolint:revive // Allow fatal in examples.
	}

	creds, err := p.GetCredentials(context.Background(), "img.v1beta1/abc/def:v1.2.3", nil)
	if err != nil {
		log.Fatal(err) //nolint:revive // Allow fatal in examples.
	}

	if err := json.NewEncoder(os.Stdout).Encode(creds); err != nil {
		log.Fatal(err) //nolint:revive // Allow fatal in examples.
	}
	//nolint:lll // Long example output.
	// Output: {"kind":"CredentialProviderResponse","apiVersion":"credentialprovider.kubelet.k8s.io/v1beta1","cacheKeyType":"Image","cacheDuration":"5s","auth":{"img.v1beta1/abc/def:v1.2.3":{"username":"v1beta1user","password":"v1beta1password"}}}
}

func ExampleCredentialProvider_GetCredentials_withMirror() {
	p, err := NewProviderFromConfigFile(filepath.Join("testdata", "config-with-mirror.yaml"))
	if err != nil {
		log.Fatal(err) //nolint:revive // Allow fatal in examples.
	}

	creds, err := p.GetCredentials(context.Background(), "img.v1beta1/abc/def:v1.2.3", nil)
	if err != nil {
		log.Fatal(err) //nolint:revive // Allow fatal in examples.
	}

	if err := json.NewEncoder(os.Stdout).Encode(creds); err != nil {
		log.Fatal(err) //nolint:revive // Allow fatal in examples.
	}
	//nolint:lll // Long example output.
	// Output: {"kind":"CredentialProviderResponse","apiVersion":"credentialprovider.kubelet.k8s.io/v1beta1","cacheKeyType":"Image","cacheDuration":"5s","auth":{"*.*":{"username":"v1beta1user","password":"v1beta1password"},"img.v1beta1/abc/def:v1.2.3":{"username":"mirroruser","password":"mirrorpassword"}}}
}
