package utils

import (
	"errors"
	"fmt"
	"io"

	"os"
	"path/filepath"
	"strings"
)

func SaveToFile(p string, cnt []byte, cover bool) (err error) {
	if string(cnt) == "" {
		return errors.New("write file: empty content")
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}
	_, err = os.Stat(p)
	if err == nil {
		if !cover {
			return fmt.Errorf("file [%s] existed, please rename or remove it", p)
		}
	}
	return WriteFileAtomic(p, cnt, 0o644)
}

// WriteFileAtomic writes content through a same-directory temporary file, so a
// failed generator run cannot leave a partially written source artifact.
func WriteFileAtomic(path string, content []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".nr-*")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return fmt.Errorf("write temporary file: %w", err)
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return fmt.Errorf("set temporary file mode: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync temporary file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace file atomically: %w", err)
	}
	return nil
}

func RemoveExt(filename string) string {
	base := filepath.Base(filename)
	ext := filepath.Ext(filename)

	return strings.Replace(base, ext, "", 1)
}
func CheckFile(fp string) error {
	info, err := os.Stat(fp)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return errors.New("path is dir")
	}

	return nil
}
func CheckFolder(fp string) error {
	info, err := os.Stat(fp)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return errors.New("path is file")
	}

	return nil
}
func CopyDir(src, dst string) error {
	// Check if source directory exists
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !srcInfo.IsDir() {
		return errors.New("source is not a directory")
	}

	// Create destination directory if it doesn't exist
	_, err = os.Stat(dst)
	if os.IsNotExist(err) {
		err = os.MkdirAll(dst, srcInfo.Mode())
		if err != nil {
			return err
		}
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectories
			err = CopyDir(srcPath, dstPath)
			if err != nil {
				return err
			}
		} else {
			// Copy files
			err = CopyFile(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	err = os.Chmod(dst, srcInfo.Mode())
	if err != nil {
		return err
	}

	return nil
}
