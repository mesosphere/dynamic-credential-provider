// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
)

type Cluster interface {
	Delete(ctx context.Context) error
}
