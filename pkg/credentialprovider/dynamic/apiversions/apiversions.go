// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Copyright 2025 Nutanix. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package apiversions

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	credentialproviderv1 "k8s.io/kubelet/pkg/apis/credentialprovider/v1"
	credentialproviderv1alpha1 "k8s.io/kubelet/pkg/apis/credentialprovider/v1alpha1"
	credentialproviderv1beta1 "k8s.io/kubelet/pkg/apis/credentialprovider/v1beta1"
)

var apiVersions = map[string]schema.GroupVersion{
	credentialproviderv1alpha1.SchemeGroupVersion.String(): credentialproviderv1alpha1.SchemeGroupVersion,
	credentialproviderv1beta1.SchemeGroupVersion.String():  credentialproviderv1beta1.SchemeGroupVersion,
	credentialproviderv1.SchemeGroupVersion.String():       credentialproviderv1.SchemeGroupVersion,
}

func APIVersions() map[string]schema.GroupVersion {
	return apiVersions
}
