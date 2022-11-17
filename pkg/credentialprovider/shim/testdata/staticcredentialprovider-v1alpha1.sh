#!/usr/bin/env bash

# Copyright 2022 D2iQ, Inc. All rights reserved.
# SPDX-License-Identifier: Apache-2.0

>&2 echo "Received Request: "
>&2 echo "$(</dev/stdin)"

echo '{
  "kind":"CredentialProviderResponse",
  "apiVersion":"credentialprovider.kubelet.k8s.io/v1alpha1",
  "cacheKeyType":"Image",
  "cacheDuration":"10s",
  "auth":{
    "registry:5000": {"username":"v1alpha1user","password":"v1alpha1password"}
  }
}'
