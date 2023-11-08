package downloader

import "os"

// IsAlreadyDownloaded checks if the target path exists.
func IsAlreadyDownloaded(target string) (bool, error) {
	_, err := os.Stat(target)

	// no error means the path exists
	if err == nil {
		return true, nil
	}

	// if the error is that the path doesn't exist, then it's not downloaded
	if os.IsNotExist(err) {
		return false, nil
	}

	// There was some other error, like a permission problem. Report it.
	return false, err
}
