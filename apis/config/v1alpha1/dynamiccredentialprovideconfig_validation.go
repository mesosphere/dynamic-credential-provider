// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var _ webhook.Validator = &DynamicCredentialProviderConfig{}

// ValidateCreate implements webhook.Validator so a webhook can be registered for the type.
func (c *DynamicCredentialProviderConfig) ValidateCreate() (admission.Warnings, error) {
	return c.validate()
}

// ValidateUpdate implements webhook.Validator so a webhook can be registered for the type.
func (c *DynamicCredentialProviderConfig) ValidateUpdate(
	_ runtime.Object,
) (admission.Warnings, error) {
	return c.validate()
}

// ValidateDelete implements webhook.Validator so a webhook can be registered for the type.
func (*DynamicCredentialProviderConfig) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}

func (c *DynamicCredentialProviderConfig) validate() (admission.Warnings, error) {
	var allErrs field.ErrorList

	allErrs = append(
		allErrs,
		validateCredentialProviderConfig(
			c.CredentialProviders,
			field.NewPath("credentialProviders"),
		)...,
	)

	if len(allErrs) == 0 {
		return nil, nil
	}

	return nil, errors.NewInvalid(
		GroupVersion.WithKind("DynamicCredentialProviderConfig").GroupKind(),
		c.Name,
		allErrs,
	)
}
