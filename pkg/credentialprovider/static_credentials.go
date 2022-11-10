// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package credentialprovider

import (
	"context"
	"fmt"
	"os"

	"k8s.io/kubelet/pkg/apis/credentialprovider/v1alpha1"
)

// staticProvider implements the credential provider interface, reading a credentials file on the disk.
type staticProvider struct {
	credentialsFile string
}

// NewStaticProvider creates a new instance of the static credentials provider.
func NewStaticProvider(credentialsFile string) (CredentialProvider, error) {
	return staticProvider{credentialsFile: credentialsFile}, nil
}

// GetCredentials will ignore the image and args arguments and simply read a credentials file and return its content.
func (s staticProvider) GetCredentials(
	ctx context.Context,
	image string,
	args []string,
) (response *v1alpha1.CredentialProviderResponse, err error) {
	credentials, err := os.ReadFile(s.credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("error reading credentials file: %w", err)
	}

	return decodeResponse(credentials)
}

func decodeResponse(data []byte) (*v1alpha1.CredentialProviderResponse, error) {
	obj, gvk, err := codecs.UniversalDecoder(v1alpha1.SchemeGroupVersion).Decode(data, nil, nil)
	if err != nil {
		return nil, err
	}

	if gvk.Kind != "CredentialProviderResponse" {
		return nil, fmt.Errorf("kind was %q, expected CredentialProviderResponse", gvk.Kind)
	}

	if gvk.Group != v1alpha1.GroupName {
		return nil, fmt.Errorf("group was %q, expected %s", gvk.Group, v1alpha1.GroupName)
	}

	response, ok := obj.(*v1alpha1.CredentialProviderResponse)
	if !ok {
		return nil, fmt.Errorf("unable to convert %T to *CredentialProviderResponse", obj)
	}

	return response, nil
}
