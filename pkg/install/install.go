// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/kelseyhightower/envconfig"
	"github.com/otiai10/copy"
	"github.com/sirupsen/logrus"
	"go.etcd.io/etcd/client/pkg/v3/fileutil"
)

//nolint:gosec // None of these are security credentials.
const (
	CredentialProviderSourceDirEnvVar      = "CREDENTIAL_PROVIDER_SOURCE_DIR"
	CredentialProviderTargetDirEnvVar      = "CREDENTIAL_PROVIDER_TARGET_DIR"
	SkipCredentialProviderBinariesEnvVar   = "SKIP_CREDENTIAL_PROVIDER_BINARIES"
	UpdateCredentialProviderBinariesEnvVar = "UPDATE_CREDENTIAL_PROVIDER_BINARIES"
)

type config struct {
	// SkipCredentialProviderBinaries is a comma-separated list of binaries. Each binary in the list.
	// will be skipped when installing to the host.
	SkipCredentialProviderBinaries []string `envconfig:"SKIP_CREDENTIAL_PROVIDER_BINARIES"`

	// UpdateCredentialProviderBinaries controls whether or not to overwrite any binaries with the same name.
	// on the host.
	UpdateCredentialProviderBinaries bool `envconfig:"UPDATE_CREDENTIAL_PROVIDER_BINARIES" default:"true"`

	// CredentialProviderSourceDir controls where to read the binaries from to copy to the host.
	//nolint:lll // Long struct tag value.
	CredentialProviderSourceDir string `envconfig:"CREDENTIAL_PROVIDER_SOURCE_DIR" default:"/opt/image-credential-provider/bin/"`

	// CredentialProviderTargetDir controls where to place the binaries on the host.
	//nolint:lll // Long struct tag value.
	CredentialProviderTargetDir string `envconfig:"CREDENTIAL_PROVIDER_TARGET_DIR" default:"/host/etc/kubernetes/image-credential-provider/"`
}

func (c config) skipBinary(binary string) bool {
	for _, name := range c.SkipCredentialProviderBinaries {
		if name == binary {
			return true
		}
	}
	return false
}

func fileExists(file string) (bool, error) {
	info, err := os.Stat(file)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to check if file %q exists: %w", file, err)
	}
	return !info.IsDir(), nil
}

func loadConfig() (config, error) {
	var c config
	err := envconfig.Process("", &c)
	if err != nil {
		return config{}, fmt.Errorf("failed to parse env config: %w", err)
	}

	return c, nil
}

func Install(logger logrus.FieldLogger) error {
	// Load config.
	c, err := loadConfig()
	if err != nil {
		return err
	}

	if err := ensureDirExists(c.CredentialProviderTargetDir); err != nil {
		return fmt.Errorf("failed to ensure target directory exists and is writable: %w", err)
	}

	// Iterate through each binary we might want to install.
	files, err := os.ReadDir(c.CredentialProviderSourceDir)
	if err != nil {
		return fmt.Errorf("failed to read source binaries: %w", err)
	}
	for _, binary := range files {
		target := path.Join(c.CredentialProviderTargetDir, binary.Name())
		source := path.Join(c.CredentialProviderSourceDir, binary.Name())
		if c.skipBinary(binary.Name()) {
			continue
		}
		exists, err := fileExists(target)
		if err != nil {
			return err
		}
		if exists && !c.UpdateCredentialProviderBinaries {
			logger.Infof("Skipping installation of %s", target)
			continue
		}
		if err := copyFileAndPermissions(source, target, logger); err != nil {
			return fmt.Errorf("failed to install %q: %w", target, err)
		}
		logger.Infof("Installed %s", target)
	}

	logger.Infof("Wrote credential provider binaries to %s\n", c.CredentialProviderTargetDir)

	return nil
}

func ensureDirExists(dir string) error {
	// Create the directory if it does not already exist.
	//nolint:revive // 755 is easy to understand file permissions.
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Ensure the directory is writeable.
	if err := fileutil.IsDirWriteable(dir); err != nil {
		return fmt.Errorf("directory is not writeable: %w", err)
	}

	return nil
}

// copyFileAndPermissions copies file permission.
func copyFileAndPermissions(src, dst string, logger logrus.FieldLogger) error {
	// If the source and destination are the same, we can simply return.
	skip, err := destinationUpToDate(src, dst, logger)
	if err != nil {
		return err
	}
	if skip {
		logger.WithField("file", dst).Info("File is already up to date, skipping")
		return nil
	}

	// Make a temporary file at the destination.
	dstTmp := fmt.Sprintf("%s.tmp", dst)
	if err := copy.Copy(src, dstTmp); err != nil {
		return fmt.Errorf("failed to copy file: %s", err)
	}

	// Move the temporary file into position. Using Rename is atomic
	// (i.e., mv) and avoids issues where the destination file is in use.
	err = os.Rename(dstTmp, dst)
	if err != nil {
		return fmt.Errorf("failed to rename file: %s", err)
	}

	return nil
}

// destinationUptoDate compares the given files and returns
// whether or not the destination file needs to be updated with the
// contents of the source file.
//
//nolint:revive // Easy to read func that just does a lot, ignore cyclomatic complexity.
func destinationUpToDate(src, dst string, logger logrus.FieldLogger) (bool, error) {
	// Stat the src file.
	f1info, err := os.Stat(src)
	if os.IsNotExist(err) {
		// If the source file doesn't exist, that's an unrecoverable problem.
		return false, err
	} else if err != nil {
		return false, err
	}

	// Stat the dst file.
	f2info, err := os.Stat(dst)
	if os.IsNotExist(err) {
		// If the destination file doesn't exist, it means the
		// two files are not equal.
		return false, nil
	} else if err != nil {
		return false, err
	}

	// First, compare the files sizes and modes. No point in comparing
	// file contents if they differ in size or file mode.
	if f1info.Size() != f2info.Size() {
		return false, nil
	}
	if f1info.Mode() != f2info.Mode() {
		return false, nil
	}

	// Files have the same exact size and mode, check the actual contents.
	f1, err := os.Open(src)
	if err != nil {
		logger.Fatal(err)
	}
	defer f1.Close()

	f2, err := os.Open(dst)
	if err != nil {
		return false, err
	}
	defer f2.Close()

	// Create a buffer, which we'll use to read both files.
	buf := make([]byte, 64000)

	// Iterate the files until we reach the end. If we spot a difference,
	// we know that the files are not the same. Otherwise, if we reach the
	// end of the file before seeing a difference, the files are identical.
	for {
		// Read the two files.
		bytesRead, err1 := f1.Read(buf)
		s1 := string(buf[:bytesRead])
		bytesRead2, err2 := f2.Read(buf)
		s2 := string(buf[:bytesRead2])

		if err1 != nil || err2 != nil {
			switch {
			case err1 == io.EOF && err2 == io.EOF:
				// Reached the end of both files.
				return true, nil
			case err1 == io.EOF || err2 == io.EOF:
				// Reached the end of one file, but not the other. They are different.
				return false, nil
			case err1 != nil:
				// Other error - return it.
				return false, err1
			case err2 != nil:
				// Other error - return it.
				return false, err2
			case bytesRead != bytesRead2:
				// Read a different number of bytes from each file. Defensively
				// consider the files different.
				return false, nil
			}
		}

		if s1 != s2 {
			// The slice of bytes we read from each file are not equal.
			return false, nil
		}
	}
}
