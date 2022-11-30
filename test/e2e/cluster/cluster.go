// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Cluster interface {
	Delete(ctx context.Context) error

	RESTConfig() *rest.Config
	Client() kubernetes.Interface
	DynamicClient() dynamic.Interface

	ControllerRuntimeClient(opts client.Options) (client.Client, error)
}
