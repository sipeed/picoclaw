package utils

import "os"

// WritePrivateFile writes data and enforces 0600 permissions for both new and existing files.
func WritePrivateFile(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}
	return os.Chmod(path, 0600)
}
