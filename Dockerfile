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

# hadolint ignore=DL3059
RUN --mount=type=bind,src=credential-providers,target=/go/src/credential-providers \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOARCH=${TARGETARCH} \
        go build -trimpath -ldflags="-s -w" \
        -o /go/bin/gcr-credential-provider \
        k8s.io/cloud-provider-gcp/cmd/auth-provider-gcp

# hadolint ignore=DL3029
FROM --platform=linux/amd64 gcr.io/distroless/static@sha256:931e1a0e48addb212ec22efc21dc63a71568a1a5609764cb587f1e383b350f28 as linux-amd64
# hadolint ignore=DL3029
FROM --platform=linux/arm64 gcr.io/distroless/static@sha256:077acdf091bcc6faa7bec809f46c2f8ddbfb311613ca2ccd266342b9d8711530 as linux-arm64

# hadolint ignore=DL3006,DL3029
FROM --platform=linux/${TARGETARCH} linux-${TARGETARCH}

COPY --from=credential_provider_builder \
     /go/bin/ecr-credential-provider \
     /go/bin/acr-credential-provider \
     /go/bin/gcr-credential-provider \
     /opt/image-credential-provider/bin/
COPY static-credential-provider dynamic-credential-provider /opt/image-credential-provider/bin/

ENTRYPOINT ["/opt/image-credential-provider/bin/dynamic-credential-provider"]
