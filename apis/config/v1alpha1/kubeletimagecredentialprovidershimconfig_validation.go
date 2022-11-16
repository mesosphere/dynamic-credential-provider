// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/errors"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var _ webhook.Validator = &KubeletImageCredentialProviderShimConfig{}

// ValidateCreate implements webhook.Validator so a webhook can be registered for the type.
func (c *KubeletImageCredentialProviderShimConfig) ValidateCreate() error {
	return c.validate()
}

// ValidateUpdate implements webhook.Validator so a webhook can be registered for the type.
func (c *KubeletImageCredentialProviderShimConfig) ValidateUpdate(_ runtime.Object) error {
	return c.validate()
}

// ValidateDelete implements webhook.Validator so a webhook can be registered for the type.
func (*KubeletImageCredentialProviderShimConfig) ValidateDelete() error {
	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

func (c *KubeletImageCredentialProviderShimConfig) validate() error {
	var allErrs field.ErrorList

	allErrs = append(
		allErrs,
		validateCredentialProviderConfig(
			c.CredentialProviders,
			field.NewPath("credentialProviders"),
		)...,
	)

	if len(allErrs) == 0 {
		return nil
	}

	return errors.NewInvalid(
		GroupVersion.WithKind("KubeletImageCredentialProviderShimConfig").GroupKind(),
		c.Name,
		allErrs,
	)
}
