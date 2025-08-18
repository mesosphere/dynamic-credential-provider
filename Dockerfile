# Copyright 2022 D2iQ, Inc. All rights reserved.
# SPDX-License-Identifier: Apache-2.0

# syntax=docker/dockerfile:1

ARG GO_VERSION
# hadolint ignore=DL3029
FROM --platform=linux/${BUILDARCH} golang:${GO_VERSION} AS credential_provider_builder

ARG TARGETARCH

WORKDIR /go/src/credential-providers
RUN --mount=type=bind,src=credential-providers,target=/go/src/credential-providers \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOARCH=${TARGETARCH} \
        go build -trimpath -ldflags="-s -w" \
        -o /go/bin/ecr-credential-provider \
        k8s.io/cloud-provider-aws/cmd/ecr-credential-provider

# hadolint ignore=DL3059
RUN --mount=type=bind,src=credential-providers,target=/go/src/credential-providers \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOARCH=${TARGETARCH} \
        go build -trimpath -ldflags="-s -w" \
        -o /go/bin/acr-credential-provider \
        sigs.k8s.io/cloud-provider-azure/cmd/acr-credential-provider

# hadolint ignore=DL3059
RUN --mount=type=bind,src=credential-providers,target=/go/src/credential-providers \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOARCH=${TARGETARCH} \
        go build -trimpath -ldflags="-s -w" \
        -o /go/bin/gcr-credential-provider \
        k8s.io/cloud-provider-gcp/cmd/auth-provider-gcp

# hadolint ignore=DL3029
FROM --platform=linux/amd64 gcr.io/distroless/static@sha256:43719ef69e9adb4d3458ee45c633282a3d13f1b4ebc1859adddfe4aade3a9ac7 AS linux-amd64
# hadolint ignore=DL3029
FROM --platform=linux/arm64 gcr.io/distroless/static@sha256:caf2029bbcfd4b4f60ea5406ee05df56a1bc966977cf6efff6f36906b10e3bc6 AS linux-arm64

# hadolint ignore=DL3006,DL3029
FROM --platform=linux/${TARGETARCH} linux-${TARGETARCH}

COPY --from=credential_provider_builder \
     /go/bin/ecr-credential-provider \
     /go/bin/acr-credential-provider \
     /go/bin/gcr-credential-provider \
     /opt/image-credential-provider/bin/
COPY static-credential-provider dynamic-credential-provider /opt/image-credential-provider/bin/

ENTRYPOINT ["/opt/image-credential-provider/bin/dynamic-credential-provider"]
