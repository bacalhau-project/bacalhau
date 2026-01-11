//nolint:unused
package migrations

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func getViper(r repo.FsRepo) (*viper.Viper, error) {
	repoPath, err := r.Path()
	if err != nil {
		return nil, err
	}

	configFile := filepath.Join(repoPath, config.DefaultFileName)
	v := viper.New()
	v.SetTypeByDefaultValue(true)
	v.SetConfigFile(configFile)

	// read existing config file if it exists
	if err := v.ReadInConfig(); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	return v, nil
}

func configExists(r repo.FsRepo) (bool, error) {
	repoPath, err := r.Path()
	if err != nil {
		return false, err
	}

	configFile := filepath.Join(repoPath, config.DefaultFileName)
	_, err = os.Stat(configFile)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func readConfig(r repo.FsRepo) (*viper.Viper, types.Bacalhau, error) {
	v, err := getViper(r)
	if err != nil {
		return nil, types.Bacalhau{}, err
	}
	var fileCfg types.Bacalhau
	if err := v.Unmarshal(&fileCfg, config.DecoderHook); err != nil {
		return v, types.Bacalhau{}, fmt.Errorf("failed to unmarshal config file: %w", err)
	}
	return v, fileCfg, nil
}

// copyFile copies a file from srcPath to dstPath, preserving the file permissions.
// It opens the source file, creates the destination file, copies the content,
// and sets the destination file's permissions to match the source file. It also
// ensures that the destination file's contents are flushed to disk before returning.
func copyFile(srcPath, dstPath string) error {
	// Open the source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	// Get the file info of the source file to retrieve its permissions
	srcFileInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	// Create the destination file
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer func() { _ = dstFile.Close() }()

	// Copy the contents from source to destination
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	// Set the destination file's permissions to match the source file's permissions
	err = os.Chmod(dstPath, srcFileInfo.Mode())
	if err != nil {
		return err
	}

	// Flush the contents to disk
	err = dstFile.Sync()
	if err != nil {
		return err
	}

	return nil
}

// writeInstallationID writes the installation ID to system wide config path
func writeInstallationID(cfg system.GlobalConfig, installationID string) error {
	// Create config dir if it doesn't exist
	if err := os.MkdirAll(cfg.ConfigDir(), util.OS_USER_RW); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	// Write installation ID to file
	installationIDFile := filepath.Join(cfg.ConfigDir(), system.InstallationIDFile)
	if err := os.WriteFile(installationIDFile, []byte(installationID), util.OS_USER_RW); err != nil {
		return fmt.Errorf("writing installation ID file: %w", err)
	}
	return nil
}
