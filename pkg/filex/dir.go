package filex

import (
	"os"
	"path/filepath"
)

// LoadFiles load files with filter func
func LoadFiles(dir string, filter func(string) bool) ([]string, error) {
	files := make([]string, 0)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filter(path) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}
