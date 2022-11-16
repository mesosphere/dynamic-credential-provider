// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

/*
Copyright 2014 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// The content of this file is copied from
// https://github.com/kubernetes/kubernetes/blob/v1.25.4/pkg/credentialprovider/keyring.go#L160-L233.
// Copied rather than imported as go mod because it lives in k8s.io/kubernetes which would pull in
// lots of unnecessary dependencies.

package urlglobber

import (
	"net"
	"net/url"
	"path/filepath"
	"strings"
)

// ParseSchemelessURL parses a schemeless url and returns a url.URL
// url.Parse require a scheme, but ours don't have schemes.  Adding a
// scheme to make url.Parse happy, then clear out the resulting scheme.
func ParseSchemelessURL(schemelessURL string) (*url.URL, error) {
	parsed, err := url.Parse("https://" + schemelessURL)
	if err != nil {
		return nil, err
	}
	// clear out the resulting scheme
	parsed.Scheme = ""
	return parsed, nil
}

// SplitURL splits the host name into parts, as well as the port.
func SplitURL(u *url.URL) (parts []string, port string) {
	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		// could not parse port
		host, port = u.Host, ""
	}
	return strings.Split(host, "."), port
}

// URLsMatchStr is wrapper for URLsMatch, operating on strings instead of URLs.
func URLsMatchStr(glob, target string) (bool, error) {
	globURL, err := ParseSchemelessURL(glob)
	if err != nil {
		return false, err
	}
	targetURL, err := ParseSchemelessURL(target)
	if err != nil {
		return false, err
	}
	return URLsMatch(globURL, targetURL)
}

// URLsMatch checks whether the given target url matches the glob url, which may have
// glob wild cards in the host name.
//
// Examples:
//
//	globURL=*.docker.io, targetURL=blah.docker.io => match
//	globURL=*.docker.io, targetURL=not.right.io   => no match
//
// Note that we don't support wildcards in ports and paths yet.
func URLsMatch(globURL, targetURL *url.URL) (bool, error) {
	globURLParts, globPort := SplitURL(globURL)
	targetURLParts, targetPort := SplitURL(targetURL)
	if globPort != targetPort {
		// port doesn't match
		return false, nil
	}
	if len(globURLParts) != len(targetURLParts) {
		// host name does not have the same number of parts
		return false, nil
	}
	if !strings.HasPrefix(targetURL.Path, globURL.Path) {
		// the path of the credential must be a prefix
		return false, nil
	}
	for k, globURLPart := range globURLParts {
		targetURLPart := targetURLParts[k]
		matched, err := filepath.Match(globURLPart, targetURLPart)
		if err != nil {
			return false, err
		}
		if !matched {
			// glob mismatch for some part
			return false, nil
		}
	}
	// everything matches
	return true, nil
}
