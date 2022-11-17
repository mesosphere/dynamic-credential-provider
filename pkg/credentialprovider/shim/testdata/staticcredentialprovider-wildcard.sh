#!/usr/bin/env bash

# Copyright 2022 D2iQ, Inc. All rights reserved.
# SPDX-License-Identifier: Apache-2.0

>&2 echo "Received Request: "
>&2 echo "$(</dev/stdin)"

echo '{
  "kind":"CredentialProviderResponse",
  "apiVersion":"credentialprovider.kubelet.k8s.io/v1beta1",
  "cacheKeyType":"Image",
  "cacheDuration":"5s",
  "auth":{
    "*.*": {"username":"wildcarduser","password":"wildcardpassword"}
  }
}'
