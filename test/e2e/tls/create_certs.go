// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

//nolint:revive // A long and repetitive function.
func GenerateCertificates(hostnames ...string) (dir string, cleanup func() error, err error) {
	dir, err = os.MkdirTemp("", "temp-tls-*")
	if err != nil {
		return "", nil, err
	}
	cleanup = func() error { return os.RemoveAll(dir) }

	ca := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"DUMMY"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, 1),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// create our private and public key
	caPrivKey, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		_ = cleanup()
		return "", nil, err
	}

	// create the CA
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		_ = cleanup()
		return "", nil, err
	}

	// pem encode
	caFile, err := os.Create(filepath.Join(dir, "ca.crt"))
	if err != nil {
		_ = cleanup()
		return "", nil, err
	}
	defer caFile.Close()
	if err := pem.Encode(caFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	}); err != nil {
		_ = caFile.Close()
		_ = cleanup()
		return "", nil, err
	}

	encodedCAPrivKey, err := x509.MarshalECPrivateKey(caPrivKey)
	if err != nil {
		return "", nil, err
	}
	caKey, err := os.Create(filepath.Join(dir, "ca.key"))
	if err != nil {
		_ = cleanup()
		return "", nil, err
	}
	defer caKey.Close()
	if err := pem.Encode(caKey, &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: encodedCAPrivKey,
	}); err != nil {
		_ = caKey.Close()
		_ = cleanup()
		return "", nil, err
	}

	// set up our server certificate
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"DUMMY"},
		},
		DNSNames:    hostnames,
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(0, 0, 1),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certPrivKey, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		_ = cleanup()
		return "", nil, err
	}

	certBytes, err := x509.CreateCertificate(
		rand.Reader,
		cert,
		ca,
		&certPrivKey.PublicKey,
		caPrivKey,
	)
	if err != nil {
		_ = cleanup()
		return "", nil, err
	}

	certFile, err := os.Create(filepath.Join(dir, "tls.crt"))
	if err != nil {
		_ = cleanup()
		return "", nil, err
	}
	defer certFile.Close()
	if err := pem.Encode(certFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}); err != nil {
		_ = certFile.Close()
		_ = cleanup()
		return "", nil, err
	}

	encodedCertPrivKey, err := x509.MarshalECPrivateKey(certPrivKey)
	if err != nil {
		_ = cleanup()
		return "", nil, err
	}
	keyFile, err := os.Create(filepath.Join(dir, "tls.key"))
	if err != nil {
		_ = cleanup()
		return "", nil, err
	}
	defer keyFile.Close()
	if err := pem.Encode(keyFile, &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: encodedCertPrivKey,
	}); err != nil {
		_ = keyFile.Close()
		_ = cleanup()
		return "", nil, err
	}

	return dir, cleanup, nil
}
