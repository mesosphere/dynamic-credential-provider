// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import "sigs.k8s.io/controller-runtime/pkg/webhook"

var _ webhook.Defaulter = &DynamicCredentialProviderConfig{}

func (c *DynamicCredentialProviderConfig) Default() {
	SetObjectDefaults_DynamicCredentialProviderConfig(c)
}

//nolint:revive,stylecheck // The underscore naming is required for kubernetes defaulter-gen.
func SetDefaults_MirrorConfig(obj *MirrorConfig) {
	if obj.MirrorCredentialsStrategy == "" {
		obj.MirrorCredentialsStrategy = MirrorCredentialsOnly
	}
}
