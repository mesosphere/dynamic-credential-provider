// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package install_test

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mesosphere/dynamic-credential-provider/pkg/install"
)

func hashFile(a string) (string, error) {
	h := sha256.New()

	f, err := os.Open(a)
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, err = io.Copy(h, f)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func assertFileHashesEqual(t *testing.T, a, b string) {
	t.Helper()

	aHash, err := hashFile(a)
	require.NoError(t, err)
	bHash, err := hashFile(b)
	require.NoError(t, err)

	assert.Equal(t, aHash, bHash)
}

func assertFileHashesDifferent(t *testing.T, a, b string) {
	t.Helper()

	aHash, err := hashFile(a)
	require.NoError(t, err)
	bHash, err := hashFile(b)
	require.NoError(t, err)

	assert.NotEqual(t, aHash, bHash)
}

func TestSuccessfulCopy(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv(install.CredentialProviderSourceDirEnvVar, "testdata")
	t.Setenv(install.CredentialProviderTargetDirEnvVar, tmpDir)

	require.NoError(t, install.Install(logrus.New()))

	testFiles, err := os.ReadDir("testdata")
	require.NoError(t, err)

	for _, f := range testFiles {
		expectedFile := filepath.Join(tmpDir, f.Name())
		assert.FileExists(t, expectedFile)
		srcFile := filepath.Join("testdata", f.Name())
		srcFileStat, err := os.Stat(srcFile)
		require.NoError(t, err)
		expectedFileStat, err := os.Stat(expectedFile)
		require.NoError(t, err)
		assert.Equal(t, srcFileStat.Mode(), expectedFileStat.Mode())
		assertFileHashesEqual(t, srcFile, expectedFile)
	}
}

func TestSuccessfulCopyNonExistentTarget(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "nonexistent")

	t.Setenv(install.CredentialProviderSourceDirEnvVar, "testdata")
	t.Setenv(install.CredentialProviderTargetDirEnvVar, targetDir)

	require.NoError(t, install.Install(logrus.New()))

	testFiles, err := os.ReadDir("testdata")
	require.NoError(t, err)

	for _, f := range testFiles {
		expectedFile := filepath.Join(targetDir, f.Name())
		assert.FileExists(t, expectedFile)
		srcFile := filepath.Join("testdata", f.Name())
		srcFileStat, err := os.Stat(srcFile)
		require.NoError(t, err)
		expectedFileStat, err := os.Stat(expectedFile)
		require.NoError(t, err)
		assert.Equal(t, srcFileStat.Mode(), expectedFileStat.Mode())
		assertFileHashesEqual(t, srcFile, expectedFile)
	}
}

func TestSuccessfulCopySkipFile(t *testing.T) {
	tmpDir := t.TempDir()

	const dummyBinary2Name = "dummybinary2"

	t.Setenv(install.CredentialProviderSourceDirEnvVar, "testdata")
	t.Setenv(install.CredentialProviderTargetDirEnvVar, tmpDir)
	t.Setenv(install.SkipCredentialProviderBinariesEnvVar, dummyBinary2Name)

	require.NoError(t, install.Install(logrus.New()))

	testFiles, err := os.ReadDir("testdata")
	require.NoError(t, err)

	for _, f := range testFiles {
		destFile := filepath.Join(tmpDir, f.Name())

		if f.Name() == dummyBinary2Name {
			assert.NoFileExists(t, destFile)
			continue
		}

		assert.FileExists(t, destFile)
		srcFile := filepath.Join("testdata", f.Name())
		srcFileStat, err := os.Stat(srcFile)
		require.NoError(t, err)
		expectedFileStat, err := os.Stat(destFile)
		require.NoError(t, err)
		assert.Equal(t, srcFileStat.Mode(), expectedFileStat.Mode())
		assertFileHashesEqual(t, srcFile, destFile)
	}
}

func TestSuccessfulCopyDoNotUpdate(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv(install.CredentialProviderSourceDirEnvVar, "testdata")
	t.Setenv(install.CredentialProviderTargetDirEnvVar, tmpDir)
	t.Setenv(install.UpdateCredentialProviderBinariesEnvVar, "false")

	require.NoError(
		t,
		//nolint:revive // Dummy value in test file, no need for const.
		os.WriteFile(filepath.Join(tmpDir, "dummybinary2"), []byte("differentcontent"), 0o600),
	)

	require.NoError(t, install.Install(logrus.New()))

	testFiles, err := os.ReadDir("testdata")
	require.NoError(t, err)

	for _, f := range testFiles {
		srcFile := filepath.Join("testdata", f.Name())
		expectedFile := filepath.Join(tmpDir, f.Name())

		assert.FileExists(t, expectedFile)

		if f.Name() == "dummybinary2" {
			assertFileHashesDifferent(t, srcFile, expectedFile)
			continue
		}

		srcFileStat, err := os.Stat(srcFile)
		require.NoError(t, err)
		expectedFileStat, err := os.Stat(expectedFile)
		require.NoError(t, err)
		assert.Equal(t, srcFileStat.Mode(), expectedFileStat.Mode())
		assertFileHashesEqual(t, srcFile, expectedFile)
	}
}
