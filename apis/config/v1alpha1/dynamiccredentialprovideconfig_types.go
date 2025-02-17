// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

//go:generate controller-gen object paths=.

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeletconfigv1 "k8s.io/kubelet/config/v1"
)

//nolint:gochecknoinits // init is the convention for registering API types.
func init() {
	SchemeBuilder.Register(&DynamicCredentialProviderConfig{})
}

// DynamicCredentialProviderConfig holds the configuration.
// +kubebuilder:object:root=true
type DynamicCredentialProviderConfig struct {
	//nolint:revive // inline is not an official json struct tag value, but is required by Kubernetes.
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"` //nolint:tagliatelle // This is the upstream convention.

	// Mirror is the optional mirror configuration.
	// +optional
	Mirror *MirrorConfig `json:"mirror,omitempty"`

	// CredentialProviders holds the configuration for the kubelet credential providers. Embeds the
	// `CredentialProviderConfig` kind from kubelet config API - see
	// https://github.com/kubernetes/kubelet/blob/v0.26.3/config/v1/types.go#L28 for info.
	// +optional
	CredentialProviders *kubeletconfigv1.CredentialProviderConfig `json:"credentialProviders,omitempty"`

	// CredentialProviderPluginBinDir is the directory where credential provider plugin binaries are located.
	CredentialProviderPluginBinDir string `json:"credentialProviderPluginBinDir,omitempty"`
}

// MirrorCredentialsStrategy specifies how to handle mirror registry credentials.
// +kubebuilder:validation:Enum=MirrorCredentialsOnly;MirrorCredentialsFirst;MirrorCredentialsLast
// +kubebuilder:validation:Default=MirrorCredentialsOnly
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
	CredentialsStrategy MirrorCredentialsStrategy `json:"credentialsStrategy"`
}
