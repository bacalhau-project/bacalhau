package migrations

import (
	"encoding/base64"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	libp2p_crypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

const libp2pPrivateKey = "libp2p_private_key"

func getViper(r repo.FsRepo) (*viper.Viper, error) {
	repoPath, err := r.Path()
	if err != nil {
		return nil, err
	}

	configFile := filepath.Join(repoPath, config.FileName)
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

	configFile := filepath.Join(repoPath, config.FileName)
	_, err = os.Stat(configFile)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func readConfig(r repo.FsRepo) (*viper.Viper, types.BacalhauConfig, error) {
	v, err := getViper(r)
	if err != nil {
		return nil, types.BacalhauConfig{}, err
	}
	var fileCfg types.BacalhauConfig
	if err := v.Unmarshal(&fileCfg, config.DecoderHook); err != nil {
		return v, types.BacalhauConfig{}, fmt.Errorf("failed to unmarshal config file: %w", err)
	}
	return v, fileCfg, nil
}

// TODO: remove this with 1.5 release as we don't expect users to migrate from 1.2 to 1.5
func getLibp2pNodeID(repoPath string) (string, error) {
	path := filepath.Join(repoPath, libp2pPrivateKey)
	privKey, err := loadLibp2pPrivKey(path)
	if err != nil {
		return "", err
	}
	peerID, err := peer.IDFromPrivateKey(privKey)
	if err != nil {
		return "", err
	}
	return peerID.String(), nil
}

func loadLibp2pPrivKey(path string) (libp2p_crypto.PrivKey, error) {
	keyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}
	// base64 decode keyBytes
	b64, err := base64.StdEncoding.DecodeString(string(keyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}
	// parse the private key
	key, err := libp2p_crypto.UnmarshalPrivateKey(b64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}
	return key, nil
}

// copyFS copies the file system fsys into the directory dir,
// creating dir if necessary.
//
// Newly created directories and files have their default modes
// according to the current umask, except that the execute bits
// are copied from the file in fsys when creating a local file.
//
// If a file name in fsys does not satisfy filepath.IsLocal,
// an error is returned for that file.
//
// Copying stops at and returns the first error encountered.
// Credit: https://github.com/golang/go/issues/62484
func copyFS(dir string, fsys fs.FS) error {
	return fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Handle the error before accessing d
			return fmt.Errorf("error accessing %s: %v", path, err)
		}

		targ := filepath.Join(dir, filepath.FromSlash(path))

		if d.IsDir() {
			// Create the directory
			dinfor, err := d.Info()
			if err != nil {
				return fmt.Errorf("stating directory: %w", err)
			}
			if err := os.MkdirAll(targ, dinfor.Mode()); err != nil {
				return fmt.Errorf("creating directory %s: %v", targ, err)
			}
			return nil
		}

		// Open the file in the fs.FS
		r, err := fsys.Open(path)
		if err != nil {
			return fmt.Errorf("opening file %s: %v", path, err)
		}
		defer r.Close()

		// Get file info to copy the mode
		info, err := r.Stat()
		if err != nil {
			return fmt.Errorf("getting file info for %s: %v", path, err)
		}

		// Create the destination file with the same permissions
		w, err := os.OpenFile(targ, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return fmt.Errorf("creating file %s: %v", targ, err)
		}

		// Copy the file contents
		if _, err := io.Copy(w, r); err != nil {
			w.Close() // ensure the file is closed even if there's an error
			return fmt.Errorf("copying %s: %v", path, err)
		}

		// Close the destination file
		if err := w.Close(); err != nil {
			return fmt.Errorf("closing file %s: %v", targ, err)
		}

		return nil
	})
}

func copyFile(srcPath, dstPath string) error {
	// Open the source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

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
	defer dstFile.Close()

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
