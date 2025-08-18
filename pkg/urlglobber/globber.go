// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package urlglobber

import (
	"errors"
	"fmt"
	"net"
	"regexp"

	"github.com/distribution/reference"
)

var (
	domainSegmentRE = regexp.MustCompile(`[^.]+`)

	ErrInvalidImageReference = errors.New("invalid image reference")
)

func GlobbedDomainForImage(img string) (string, error) {
	namedImg, err := reference.ParseNormalizedNamed(img)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidImageReference, err)
	}

	domain := reference.Domain(namedImg)

	domainWithoutPort, port, errSplitPort := net.SplitHostPort(domain)
	if errSplitPort == nil {
		domain = domainWithoutPort
	}
	globbedDomain := domainSegmentRE.ReplaceAllLiteralString(domain, "*")
	if errSplitPort == nil {
		globbedDomain = net.JoinHostPort(globbedDomain, port)
	}

	return globbedDomain, nil
}
