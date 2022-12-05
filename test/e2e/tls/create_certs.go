// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package tls

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

func GenerateCertificates(dir string, hostnames ...string) error {
	ca, caPrivKey, err := generateCA(dir)
	if err != nil {
		return fmt.Errorf("failed top create CA: %w", err)
	}

	if err := generateServerCerts(ca, caPrivKey, hostnames, dir); err != nil {
		return fmt.Errorf("failed to generate TLS certificates: %w", err)
	}

	return nil
}

func generateCA(outputDir string) (*x509.Certificate, *ecdsa.PrivateKey, error) {
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
		return nil, nil, fmt.Errorf("failed to generate CA private key: %w", err)
	}

	// create the CA
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to issue CA certificate: %w", err)
	}

	caFile, err := os.Create(filepath.Join(outputDir, "ca.crt"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create CA certificate file: %w", err)
	}
	defer caFile.Close()
	err = pem.Encode(caFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to PEM-encode CA certificate: %w", err)
	}

	encodedCAPrivKey, err := x509.MarshalECPrivateKey(caPrivKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal CA private key: %w", err)
	}

	caKey, err := os.Create(filepath.Join(outputDir, "ca.key"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create CA private key file: %w", err)
	}
	defer caKey.Close()
	err = pem.Encode(caKey, &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: encodedCAPrivKey,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to PEM-encode CA private key: %w", err)
	}

	return ca, caPrivKey, nil
}

func generateServerCerts(
	ca *x509.Certificate,
	caPrivKey crypto.PrivateKey,
	hostnames []string,
	outputDir string,
) error {
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
		return fmt.Errorf("failed to generate TLS private key: %w", err)
	}

	certBytes, err := x509.CreateCertificate(
		rand.Reader,
		cert,
		ca,
		&certPrivKey.PublicKey,
		caPrivKey,
	)
	if err != nil {
		return fmt.Errorf("failed to issue TLS certificate: %w", err)
	}

	certFile, err := os.Create(filepath.Join(outputDir, "tls.crt"))
	if err != nil {
		return fmt.Errorf("failed to create TLS certificate file: %w", err)
	}
	defer certFile.Close()
	err = pem.Encode(certFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	if err != nil {
		return fmt.Errorf("failed to PEM-encode TLS certificate: %w", err)
	}

	encodedCertPrivKey, err := x509.MarshalECPrivateKey(certPrivKey)
	if err != nil {
		return fmt.Errorf("failed to marshal TLS private key: %w", err)
	}
	keyFile, err := os.Create(filepath.Join(outputDir, "tls.key"))
	if err != nil {
		return fmt.Errorf("failed to create TLS private key file: %w", err)
	}
	defer keyFile.Close()
	err = pem.Encode(keyFile, &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: encodedCertPrivKey,
	})
	if err != nil {
		return fmt.Errorf("failed to create PEM-encode TLS private key: %w", err)
	}

	return err
}
