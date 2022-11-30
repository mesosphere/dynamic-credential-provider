// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package env

import "os"

func DockerHubUsername() string {
	return os.Getenv("DOCKER_USERNAME")
}

func DockerHubPassword() string {
	return os.Getenv("DOCKER_PASSWORD")
}
