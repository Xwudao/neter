package filex

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SyncDir synchronizes src to dst. It copies files whose contents differ,
// preserves empty directories, and removes entries from dst that are absent
// from src. Source and destination must be distinct, non-overlapping trees.
func SyncDir(src, dst string) error {
	srcAbs, dstAbs, err := validateSyncPaths(src, dst)
	if err != nil {
		return err
	}

	dirs := make(map[string]fs.FileMode)
	files := make(map[string]fs.FileMode)
	if err := filepath.WalkDir(srcAbs, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == srcAbs {
			return nil
		}

		rel, err := filepath.Rel(srcAbs, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		switch {
		case info.IsDir():
			dirs[rel] = info.Mode()
		case info.Mode().IsRegular():
			files[rel] = info.Mode()
		default:
			return fmt.Errorf("unsupported source entry %s (%s)", rel, info.Mode().Type())
		}
		return nil
	}); err != nil {
		return fmt.Errorf("walk source: %w", err)
	}

	if err := os.MkdirAll(dstAbs, 0755); err != nil {
		return fmt.Errorf("create destination root: %w", err)
	}

	// Process parents first so a stale file can safely be replaced by a
	// directory containing source files.
	dirPaths := make([]string, 0, len(dirs))
	for rel := range dirs {
		dirPaths = append(dirPaths, rel)
	}
	sort.Strings(dirPaths)
	for _, rel := range dirPaths {
		mode := dirs[rel]
		path := filepath.Join(dstAbs, rel)
		if info, err := os.Lstat(path); err == nil && !info.IsDir() {
			if err := os.RemoveAll(path); err != nil {
				return fmt.Errorf("replace %s with directory: %w", rel, err)
			}
		}
		if err := os.MkdirAll(path, mode.Perm()); err != nil {
			return fmt.Errorf("create directory %s: %w", rel, err)
		}
		if err := os.Chmod(path, mode.Perm()); err != nil {
			return fmt.Errorf("set directory mode %s: %w", rel, err)
		}
	}

	for rel, mode := range files {
		srcPath := filepath.Join(srcAbs, rel)
		dstPath := filepath.Join(dstAbs, rel)
		if info, err := os.Lstat(dstPath); err == nil && !info.Mode().IsRegular() {
			if err := os.RemoveAll(dstPath); err != nil {
				return fmt.Errorf("replace %s with file: %w", rel, err)
			}
		}

		same, err := sameFileContents(srcPath, dstPath)
		if err != nil {
			return fmt.Errorf("compare %s: %w", rel, err)
		}
		if same {
			if err := preserveFileMetadata(srcPath, dstPath, mode); err != nil {
				return fmt.Errorf("update metadata for %s: %w", rel, err)
			}
			continue
		}
		if err := copyFilePreserve(srcPath, dstPath, mode); err != nil {
			return fmt.Errorf("copy %s: %w", rel, err)
		}
	}

	if err := removeStale(dstAbs, dstAbs, dirs, files); err != nil {
		return fmt.Errorf("remove stale entries: %w", err)
	}
	return nil
}

func validateSyncPaths(src, dst string) (string, string, error) {
	srcAbs, err := filepath.Abs(src)
	if err != nil {
		return "", "", fmt.Errorf("resolve source: %w", err)
	}
	dstAbs, err := filepath.Abs(dst)
	if err != nil {
		return "", "", fmt.Errorf("resolve destination: %w", err)
	}
	info, err := os.Stat(srcAbs)
	if err != nil {
		return "", "", fmt.Errorf("stat source: %w", err)
	}
	if !info.IsDir() {
		return "", "", fmt.Errorf("source is not a directory: %s", src)
	}
	if dstInfo, err := os.Lstat(dstAbs); err == nil && !dstInfo.IsDir() {
		return "", "", fmt.Errorf("destination is not a directory: %s", dst)
	} else if err != nil && !os.IsNotExist(err) {
		return "", "", fmt.Errorf("stat destination: %w", err)
	}
	if pathContains(srcAbs, dstAbs) || pathContains(dstAbs, srcAbs) {
		return "", "", fmt.Errorf("source and destination must not overlap: %s, %s", src, dst)
	}
	return srcAbs, dstAbs, nil
}

func pathContains(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func removeStale(root, dir string, dirs, files map[string]fs.FileMode) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if _, exists := dirs[rel]; !exists {
				if err := os.RemoveAll(path); err != nil {
					return fmt.Errorf("remove stale directory %s: %w", rel, err)
				}
				continue
			}
			if err := removeStale(root, path, dirs, files); err != nil {
				return err
			}
			continue
		}
		if _, exists := files[rel]; !exists {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("remove stale file %s: %w", rel, err)
			}
		}
	}
	return nil
}

func sameFileContents(src, dst string) (bool, error) {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return false, err
	}
	dstInfo, err := os.Lstat(dst)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !dstInfo.Mode().IsRegular() || srcInfo.Size() != dstInfo.Size() {
		return false, nil
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return false, err
	}
	defer srcFile.Close()
	dstFile, err := os.Open(dst)
	if err != nil {
		return false, err
	}
	defer dstFile.Close()
	return filesEqual(srcFile, dstFile)
}

func filesEqual(a, b *os.File) (bool, error) {
	bufA := make([]byte, 32*1024)
	bufB := make([]byte, 32*1024)
	for {
		countA, errA := a.Read(bufA)
		countB, errB := b.Read(bufB)
		if countA != countB || !equalBytes(bufA[:countA], bufB[:countB]) {
			return false, nil
		}
		if errA == io.EOF && errB == io.EOF {
			return true, nil
		}
		if errA != nil && errA != io.EOF {
			return false, errA
		}
		if errB != nil && errB != io.EOF {
			return false, errB
		}
	}
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func copyFilePreserve(src, dst string, mode fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode.Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return preserveFileMetadata(src, dst, mode)
}

func preserveFileMetadata(src, dst string, mode fs.FileMode) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.Chmod(dst, mode.Perm()); err != nil {
		return err
	}
	return os.Chtimes(dst, info.ModTime(), info.ModTime())
}
