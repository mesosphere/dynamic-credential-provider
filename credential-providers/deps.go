// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	_ "k8s.io/cloud-provider-aws/cmd/ecr-credential-provider"
	_ "k8s.io/cloud-provider-gcp/cmd/auth-provider-gcp"
	_ "sigs.k8s.io/cloud-provider-azure/cmd/acr-credential-provider"
)
