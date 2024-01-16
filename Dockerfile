# Copyright 2022 D2iQ, Inc. All rights reserved.
# SPDX-License-Identifier: Apache-2.0

# syntax=docker/dockerfile:1

ARG GO_VERSION
# hadolint ignore=DL3029
FROM --platform=linux/${BUILDARCH} golang:${GO_VERSION} as credential_provider_builder

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

ARG CLOUD_PROVIDER_GCP_VERSION
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    git clone --depth 1 --branch ${CLOUD_PROVIDER_GCP_VERSION} --single-branch \
        https://github.com/kubernetes/cloud-provider-gcp /go/src/credential-providers && \
    CGO_ENABLED=0 GOARCH=${TARGETARCH} \
        go build -trimpath -ldflags="-s -w" \
        -o /go/bin/gcr-credential-provider \
        ./cmd/auth-provider-gcp

# hadolint ignore=DL3029
FROM --platform=linux/amd64 gcr.io/distroless/static@sha256:8758f80ad813de0232c696f6d60439182850415475c023109392c3e968070e66 as linux-amd64
# hadolint ignore=DL3029
FROM --platform=linux/arm64 gcr.io/distroless/static@sha256:82f8020461a454c498a3123a14df39563a4ae2e3522174f8fa0ad0cdd8590a65 as linux-arm64

# hadolint ignore=DL3006,DL3029
FROM --platform=linux/${TARGETARCH} linux-${TARGETARCH}

COPY --from=credential_provider_builder \
     /go/bin/ecr-credential-provider \
     /go/bin/acr-credential-provider \
     /go/bin/gcr-credential-provider \
     /opt/image-credential-provider/bin/
COPY static-credential-provider dynamic-credential-provider /opt/image-credential-provider/bin/

ENTRYPOINT ["/opt/image-credential-provider/bin/dynamic-credential-provider"]
