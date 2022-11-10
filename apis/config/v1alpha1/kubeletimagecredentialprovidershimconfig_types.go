// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

//go:generate controller-gen object paths=.
//go:generate bash -ec "cd ../../.. && defaulter-gen -h hack/boilerplate.go.txt -i ./apis/config/v1alpha1 -o ."

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//nolint:gochecknoinits // init is the convention for registering API types.
func init() {
	SchemeBuilder.Register(&KubeletImageCredentialProviderShimConfig{})
}

// KubeletImageCredentialProviderShimConfig holds the configuration.
// +kubebuilder:object:root=true
type KubeletImageCredentialProviderShimConfig struct {
	//nolint:revive // inline is not an official json struct tag value, but is required bu Kubernetes.
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Mirror is the optional mirror configuration.
	Mirror *MirrorConfig `json:"mirror,omitempty"`
}

// MirrorConfig holds the configuration for the optional registry mirror.
type MirrorConfig struct {
	// Endpoint is the registry endpoint to use for the mirror. The endpoint must be a valid url
	// with host specified. The scheme, host and path from the endpoint URL will be used.
	Endpoint string `json:"endpoint"`
	// AuthConfig is the optional static authentication configuration to use for the mirror. If
	// unspecified and the mirror endpoint is a known registry provider, the appropriate kubelet
	// image credential provider plugin will be called to retried the auth config to use.
	AuthConfig *AuthConfig `json:"authConfig,omitempty"`
}

// AuthConfig is duplicated from upstream in order to keep package dependencies local.
// AuthConfig contains authentication information for a container registry.
// Only username/password based authentication is supported today, but more authentication
// mechanisms may be added in the future.
type AuthConfig struct {
	// username is the username used for authenticating to the container registry
	// An empty username is valid.
	Username string `json:"username"`

	// password is the password used for authenticating to the container registry
	// An empty password is valid.
	Password string `json:"password"`
}
