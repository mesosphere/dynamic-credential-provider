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
	"math/big"
	"os"
	"path/filepath"
	"time"

	g "github.com/onsi/ginkgo/v2"
	gm "github.com/onsi/gomega"
)

func GenerateCertificates(hostnames ...string) string {
	dir, err := os.MkdirTemp("", "temp-tls-*")
	gm.Expect(err).NotTo(gm.HaveOccurred())
	g.DeferCleanup(os.RemoveAll, dir)

	ca, caPrivKey := generateCA(dir)

	generateServerCerts(ca, caPrivKey, hostnames, dir)

	return dir
}

func generateCA(outputDir string) (*x509.Certificate, *ecdsa.PrivateKey) {
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
	gm.Expect(err).NotTo(gm.HaveOccurred())

	// create the CA
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	gm.Expect(err).NotTo(gm.HaveOccurred())

	// pem encode
	caFile, err := os.Create(filepath.Join(outputDir, "ca.crt"))
	gm.Expect(err).NotTo(gm.HaveOccurred())
	defer caFile.Close()
	err = pem.Encode(caFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
	gm.Expect(err).NotTo(gm.HaveOccurred())

	encodedCAPrivKey, err := x509.MarshalECPrivateKey(caPrivKey)
	gm.Expect(err).NotTo(gm.HaveOccurred())

	caKey, err := os.Create(filepath.Join(outputDir, "ca.key"))
	gm.Expect(err).NotTo(gm.HaveOccurred())
	defer caKey.Close()
	err = pem.Encode(caKey, &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: encodedCAPrivKey,
	})
	gm.Expect(err).NotTo(gm.HaveOccurred())

	return ca, caPrivKey
}

func generateServerCerts(
	ca *x509.Certificate,
	caPrivKey crypto.PrivateKey,
	hostnames []string,
	outputDir string,
) {
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
	gm.Expect(err).NotTo(gm.HaveOccurred())

	certBytes, err := x509.CreateCertificate(
		rand.Reader,
		cert,
		ca,
		&certPrivKey.PublicKey,
		caPrivKey,
	)
	gm.Expect(err).NotTo(gm.HaveOccurred())

	certFile, err := os.Create(filepath.Join(outputDir, "tls.crt"))
	gm.Expect(err).NotTo(gm.HaveOccurred())
	defer certFile.Close()
	err = pem.Encode(certFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	gm.Expect(err).NotTo(gm.HaveOccurred())

	encodedCertPrivKey, err := x509.MarshalECPrivateKey(certPrivKey)
	gm.Expect(err).NotTo(gm.HaveOccurred())
	keyFile, err := os.Create(filepath.Join(outputDir, "tls.key"))
	gm.Expect(err).NotTo(gm.HaveOccurred())
	defer keyFile.Close()
	err = pem.Encode(keyFile, &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: encodedCertPrivKey,
	})
	gm.Expect(err).NotTo(gm.HaveOccurred())
}
