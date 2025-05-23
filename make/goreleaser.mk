# Copyright 2022 D2iQ, Inc. All rights reserved.
# SPDX-License-Identifier: Apache-2.0

GORELEASER_PARALLELISM ?= $(shell nproc --ignore=1)
GORELEASER_VERBOSE ?= false

ifndef GORELEASER_CURRENT_TAG
export GORELEASER_CURRENT_TAG=$(GIT_TAG)
endif

.PHONY: build-snapshot
build-snapshot: ## Builds a snapshot with goreleaser
build-snapshot: ; $(info $(M) building snapshot $*)
	goreleaser --verbose=$(GORELEASER_VERBOSE) \
		build \
		--snapshot \
		--clean \
		--parallelism=$(GORELEASER_PARALLELISM) \
		--single-target

.PHONY: release
release: ## Builds a release with goreleaser
release: ; $(info $(M) building release $*)
	goreleaser --verbose=$(GORELEASER_VERBOSE) \
		release \
		--clean \
		--parallelism=$(GORELEASER_PARALLELISM) \
		$(GORELEASER_FLAGS)

.PHONY: release-snapshot
release-snapshot: ## Builds a snapshot release with goreleaser
release-snapshot: ; $(info $(M) building snapshot release $*)
	goreleaser --verbose=$(GORELEASER_VERBOSE) \
		release \
		--snapshot \
		--clean \
		--parallelism=$(GORELEASER_PARALLELISM)
