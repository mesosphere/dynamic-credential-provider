// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// This package provides utilities to work with URL globs for credential providers.
//
// Duplicating documentation from
// https://github.com/kubernetes/kubelet/blob/v0.26.4/pkg/apis/credentialprovider/v1/types.go#L73-L101
// for visibility:
//
// auth is a map containing authentication information passed into the kubelet.
// Each key is a match image string (more on this below). The corresponding authConfig value
// should be valid for all images that match against this key. A plugin should set
// this field to null if no valid credentials can be returned for the requested image.
//
// Each key in the map is a pattern which can optionally contain a port and a path.
// Globs can be used in the domain, but not in the port or the path. Globs are supported
// as subdomains like '*.k8s.io' or 'k8s.*.io', and top-level-domains such as 'k8s.*'.
// Matching partial subdomains like 'app*.k8s.io' is also supported. Each glob can only match
// a single subdomain segment, so *.io does not match *.k8s.io.
//
// The kubelet will match images against the key when all of the below are true:
// - Both contain the same number of domain parts and each part matches.
// - The URL path of an imageMatch must be a prefix of the target image URL path.
// - If the imageMatch contains a port, then the port must match in the image as well.
//
// When multiple keys are returned, the kubelet will traverse all keys in reverse order so that:
// - longer keys come before shorter keys with the same prefix
// - non-wildcard keys come before wildcard keys with the same prefix.
//
// For any given match, the kubelet will attempt an image pull with the provided credentials,
// stopping after the first successfully authenticated pull.
//
// Example keys:
//   - 123456789.dkr.ecr.us-east-1.amazonaws.com
//   - *.azurecr.io
//   - gcr.io
//   - *.*.registry.io
//   - registry.io:8080/path
package urlglobber
