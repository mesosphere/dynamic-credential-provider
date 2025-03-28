// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

/*
Copyright 2020 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package webhooks

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"
	kubeletconfigv1 "k8s.io/kubelet/config/v1"

	"github.com/mesosphere/dynamic-credential-provider/pkg/credentialprovider/dynamic/apiversions"
	"github.com/mesosphere/dynamic-credential-provider/pkg/urlglobber"
)

// validateCredentialProviderConfig validates CredentialProviderConfig.
// Copied from https://github.com/kubernetes/kubernetes/blob/v1.25.4/pkg/credentialprovider/plugin/config.go#L72-L128.
//
//nolint:revive // This is copied as is from upstream so not refactored to reduce cyclomatic complexity.
func validateCredentialProviderConfig(
	_ context.Context,
	config *kubeletconfigv1.CredentialProviderConfig,
	rootPath *field.Path,
) field.ErrorList {
	allErrs := field.ErrorList{}

	fieldPath := rootPath.Child("providers")

	if len(config.Providers) == 0 {
		allErrs = append(
			allErrs,
			field.Required(fieldPath, "at least 1 item in plugins is required"),
		)
	}

	for idx, provider := range config.Providers {
		providerFieldPath := fieldPath.Index(idx)

		if strings.Contains(provider.Name, "/") {
			allErrs = append(
				allErrs,
				field.Invalid(
					providerFieldPath.Child("name"),
					provider.Name,
					"provider name cannot contain '/'",
				),
			)
		}

		if strings.Contains(provider.Name, " ") {
			allErrs = append(
				allErrs,
				field.Invalid(
					providerFieldPath.Child("name"),
					provider.Name,
					"provider name cannot contain spaces",
				),
			)
		}

		if provider.Name == "." {
			allErrs = append(
				allErrs,
				field.Invalid(
					providerFieldPath.Child("name"),
					provider.Name,
					"provider name cannot be '.'",
				),
			)
		}

		if provider.Name == ".." {
			allErrs = append(
				allErrs,
				field.Invalid(
					providerFieldPath.Child("name"),
					provider.Name,
					"provider name cannot be '..'",
				),
			)
		}

		if provider.APIVersion == "" {
			allErrs = append(
				allErrs,
				field.Required(providerFieldPath.Child("apiVersion"), "apiVersion is required"),
			)
		} else if _, ok := apiversions.APIVersions()[provider.APIVersion]; !ok {
			validAPIVersions := []string{}
			for apiVersion := range apiversions.APIVersions() {
				validAPIVersions = append(validAPIVersions, apiVersion)
			}

			allErrs = append(
				allErrs,
				field.NotSupported(providerFieldPath.Child("apiVersion"), provider.APIVersion, validAPIVersions),
			)
		}

		if len(provider.MatchImages) == 0 {
			allErrs = append(
				allErrs,
				field.Required(
					providerFieldPath.Child("matchImages"),
					"at least 1 item in matchImages is required",
				),
			)
		}

		for _, matchImage := range provider.MatchImages {
			if _, err := urlglobber.ParsePotentiallySchemelessURL(matchImage); err != nil {
				allErrs = append(
					allErrs,
					field.Invalid(
						providerFieldPath.Child("matchImages"),
						matchImage,
						fmt.Sprintf("match image is invalid: %s", err.Error()),
					),
				)
			}
		}

		if provider.DefaultCacheDuration == nil {
			allErrs = append(
				allErrs,
				field.Required(
					providerFieldPath.Child("defaultCacheDuration"),
					"defaultCacheDuration is required",
				),
			)
		}

		if provider.DefaultCacheDuration != nil && provider.DefaultCacheDuration.Duration < 0 {
			allErrs = append(
				allErrs,
				field.Invalid(
					providerFieldPath.Child("defaultCacheDuration"),
					provider.DefaultCacheDuration.Duration,
					"defaultCacheDuration must be greater than or equal to 0",
				),
			)
		}
	}

	return allErrs
}
