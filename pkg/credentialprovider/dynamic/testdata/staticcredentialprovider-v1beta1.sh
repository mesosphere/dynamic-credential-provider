#!/usr/bin/env bash

# Copyright 2022 D2iQ, Inc. All rights reserved.
# SPDX-License-Identifier: Apache-2.0

echo >&2 "Received Request: "
echo >&2 "$(</dev/stdin)"

echo '{
  "kind":"CredentialProviderResponse",
  "apiVersion":"credentialprovider.kubelet.k8s.io/v1beta1",
  "cacheKeyType":"Image",
  "cacheDuration":"5s",
  "auth":{
    "*.v1beta1": {"username":"v1beta1testuser","password":"v1beta1testpassword"}
  }
}'
