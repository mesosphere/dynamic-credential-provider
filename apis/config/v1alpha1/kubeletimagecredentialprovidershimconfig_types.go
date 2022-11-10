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

// +kubebuilder:object:root=true
type KubeletImageCredentialProviderShimConfig struct {
	//nolint:revive // inline is not an official json struct tag value, but is required bu Kubernetes.
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}
