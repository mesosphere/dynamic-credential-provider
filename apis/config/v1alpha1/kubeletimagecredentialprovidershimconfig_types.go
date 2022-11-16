// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

//go:generate controller-gen object paths=.
//go:generate bash -ec "cd ../../.. && defaulter-gen -h hack/boilerplate.go.txt -i ./apis/config/v1alpha1 -o ."

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubelet/config/v1beta1"
)

//nolint:gochecknoinits // init is the convention for registering API types.
func init() {
	SchemeBuilder.Register(&KubeletImageCredentialProviderShimConfig{})
}

// KubeletImageCredentialProviderShimConfig holds the configuration.
// +kubebuilder:object:root=true
type KubeletImageCredentialProviderShimConfig struct {
	//nolint:revive // inline is not an official json struct tag value, but is required by Kubernetes.
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Mirror is the optional mirror configuration.
	// +optional
	Mirror *MirrorConfig `json:"mirror,omitempty"`

	// CredentialProviders holds the configuration for the kubelet credential providers. Embeds the
	// `CredentialProviderConfig` kind from kuelet config API - see
	// https://github.com/kubernetes/kubelet/blob/v0.25.4/config/v1beta1/types.go#L921 for info.
	// +optional
	CredentialProviders *v1beta1.CredentialProviderConfig `json:"credentialProviders,omitempty"`
}

// MirrorCredentialsStrategy specifies how to handle mirror registry credentials.
// +kubebuilder:validation:Enum=MirrorCredentialsOnly;MirrorCredentialsFirst;MirrorCredentialsLast
type MirrorCredentialsStrategy string

//nolint:gosec // No credentials here.
const (
	// MirrorCredentialsFirst specifies that the mirror credentials should be first in the chain of
	// credentials to return. The credentials response should therefore contain the mirror credentials
	// with the most specific match, i.e. on the whole requested image name.
	MirrorCredentialsFirst MirrorCredentialsStrategy = "MirrorCredentialsFirst"
	// MirrorCredentialsLast specifies that the mirror credentials should be last in the chain of
	// credentials to return. The credentials response should therefore contain the mirror credentials
	// with the least specific match, i.e. on wildcards only.
	MirrorCredentialsLast MirrorCredentialsStrategy = "MirrorCredentialsLast"
	// MirrorCredentialsOnly specifies that only the mirror credentials should returned. The
	// credentials response should therefore only contain the mirror credentials for every requested
	// image.
	MirrorCredentialsOnly MirrorCredentialsStrategy = "MirrorCredentialsOnly"
)

// MirrorConfig holds the configuration for the optional registry mirror.
type MirrorConfig struct {
	// Endpoint is the registry endpoint to use for the mirror. The endpoint must be a valid url
	// with host specified. The scheme, host and path from the endpoint URL will be used.
	Endpoint string `json:"endpoint"`
	// CredentialStrategy specifies what strategy to employ when returning registry credentials.
	// Defaults to `MirrorCredentialsOnly`.
	// +optional
	MirrorCredentialsStrategy MirrorCredentialsStrategy `json:"credentialsStrategy"`
}
