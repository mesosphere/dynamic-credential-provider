# Copyright 2022 D2iQ, Inc. All rights reserved.
# SPDX-License-Identifier: Apache-2.0

REPO_ROOT := $(CURDIR)

include make/all.mk

ASDF_VERSION=v0.10.2

CI_DOCKER_BUILD_ARGS=ASDF_VERSION=$(ASDF_VERSION)
