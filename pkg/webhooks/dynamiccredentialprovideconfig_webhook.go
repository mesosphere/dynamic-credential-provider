// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package webhooks

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/mesosphere/dynamic-credential-provider/apis/config/v1alpha1"
)

type DynamicCredentialProviderConfigWebhook struct{}

var (
	_ webhook.CustomDefaulter = &DynamicCredentialProviderConfigWebhook{}
	_ webhook.CustomValidator = &DynamicCredentialProviderConfigWebhook{}
)

func (w *DynamicCredentialProviderConfigWebhook) Default(
	_ context.Context, obj runtime.Object,
) error {
	return nil
}

// ValidateCreate implements webhook.Validator so a webhook can be registered for the type.
func (w *DynamicCredentialProviderConfigWebhook) ValidateCreate(
	ctx context.Context, obj runtime.Object,
) (admission.Warnings, error) {
	return w.validate(ctx, obj)
}

// ValidateUpdate implements webhook.Validator so a webhook can be registered for the type.
func (w *DynamicCredentialProviderConfigWebhook) ValidateUpdate(
	ctx context.Context, _, obj runtime.Object,
) (admission.Warnings, error) {
	return w.validate(ctx, obj)
}

// ValidateDelete implements webhook.Validator so a webhook can be registered for the type.
func (*DynamicCredentialProviderConfigWebhook) ValidateDelete(
	_ context.Context, _ runtime.Object,
) (admission.Warnings, error) {
	return nil, nil
}

func (w *DynamicCredentialProviderConfigWebhook) validate(
	ctx context.Context, obj runtime.Object,
) (admission.Warnings, error) {
	c, ok := obj.(*v1alpha1.DynamicCredentialProviderConfig)
	if !ok {
		return nil, errors.NewBadRequest(
			fmt.Sprintf("expected a DynamicCredentialProviderConfig, got %T", obj),
		)
	}

	var allErrs field.ErrorList

	allErrs = append(
		allErrs,
		validateCredentialProviderConfig(
			ctx,
			c.CredentialProviders,
			field.NewPath("credentialProviders"),
		)...,
	)

	if len(allErrs) == 0 {
		return nil, nil
	}

	return nil, errors.NewInvalid(
		v1alpha1.GroupVersion.WithKind("DynamicCredentialProviderConfig").GroupKind(),
		c.Name,
		allErrs,
	)
}
