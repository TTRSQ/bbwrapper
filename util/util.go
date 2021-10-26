package util

import "os"

// FileExists Check file existence
func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}
