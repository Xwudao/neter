package filex

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncDir(t *testing.T) {
	// Create temp directory structure
	src := t.TempDir()
	dst := t.TempDir()

	// Create source files
	srcFiles := map[string]string{
		"index.html":     "<h1>Hello</h1>",
		"js/app.js":      "console.log('app')",
		"css/style.css":  "body { color: red }",
		"assets/img.png": "fake-png-data",
	}

	for path, content := range srcFiles {
		fullPath := filepath.Join(src, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		require.NoError(t, err)
		err = os.WriteFile(fullPath, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Run sync
	err := SyncDir(src, dst)
	require.NoError(t, err)

	// Verify all files were copied
	for path, content := range srcFiles {
		dstPath := filepath.Join(dst, path)
		assert.FileExists(t, dstPath)
		data, err := os.ReadFile(dstPath)
		require.NoError(t, err)
		assert.Equal(t, content, string(data))
	}
}

func TestSyncDir_Incremental(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Initial files
	require.NoError(t, os.WriteFile(filepath.Join(src, "a.txt"), []byte("aaa"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "b.txt"), []byte("bbb"), 0644))

	// First sync
	require.NoError(t, SyncDir(src, dst))
	assert.FileExists(t, filepath.Join(dst, "a.txt"))
	assert.FileExists(t, filepath.Join(dst, "b.txt"))

	// Add new file in src
	require.NoError(t, os.WriteFile(filepath.Join(src, "c.txt"), []byte("ccc"), 0644))

	// Modify existing file
	require.NoError(t, os.WriteFile(filepath.Join(src, "a.txt"), []byte("aaa-modified"), 0644))

	// Second sync — should only copy new/changed files
	require.NoError(t, SyncDir(src, dst))

	data, err := os.ReadFile(filepath.Join(dst, "a.txt"))
	require.NoError(t, err)
	assert.Equal(t, "aaa-modified", string(data))

	data, err = os.ReadFile(filepath.Join(dst, "c.txt"))
	require.NoError(t, err)
	assert.Equal(t, "ccc", string(data))
}

func TestSyncDir_RemoveStaleFiles(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Create files in both src and dst
	require.NoError(t, os.WriteFile(filepath.Join(src, "keep.txt"), []byte("keep"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dst, "keep.txt"), []byte("keep"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dst, "stale.txt"), []byte("stale"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(dst, "obsolete"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dst, "obsolete", "old.js"), []byte("old"), 0644))

	// Sync should remove stale files
	require.NoError(t, SyncDir(src, dst))

	assert.FileExists(t, filepath.Join(dst, "keep.txt"))
	assert.NoFileExists(t, filepath.Join(dst, "stale.txt"))
	assert.NoFileExists(t, filepath.Join(dst, "obsolete", "old.js"))
}

func TestSyncDir_EmptySrc(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dst, "should-be-removed.txt"), []byte("x"), 0644))

	require.NoError(t, SyncDir(src, dst))

	entries, err := os.ReadDir(dst)
	require.NoError(t, err)
	assert.Empty(t, entries, "dst should be empty after syncing from empty src")
}

func TestSyncDir_NestedDirectories(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Deeply nested structure
	dirs := []string{
		"a/b/c",
		"a/b/d",
		"x/y/z",
	}
	for _, d := range dirs {
		require.NoError(t, os.MkdirAll(filepath.Join(src, d), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(src, d, "file.txt"), []byte("nested"), 0644))
	}

	require.NoError(t, SyncDir(src, dst))

	for _, d := range dirs {
		assert.FileExists(t, filepath.Join(dst, d, "file.txt"))
	}
}

func TestSyncDir_SkipUnchanged(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Create identical files in both
	require.NoError(t, os.WriteFile(filepath.Join(src, "same.txt"), []byte("identical"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dst, "same.txt"), []byte("identical"), 0644))

	// Match mod times
	srcInfo, _ := os.Stat(filepath.Join(src, "same.txt"))
	_ = os.Chtimes(filepath.Join(dst, "same.txt"), srcInfo.ModTime(), srcInfo.ModTime())

	// This should be a no-op (no error)
	require.NoError(t, SyncDir(src, dst))

	data, err := os.ReadFile(filepath.Join(dst, "same.txt"))
	require.NoError(t, err)
	assert.Equal(t, "identical", string(data))
}

func TestSyncDir_ReplacesDirectoryWithFile(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "entry"), []byte("file"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(dst, "entry"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dst, "entry", "old.txt"), []byte("old"), 0644))

	require.NoError(t, SyncDir(src, dst))
	assert.FileExists(t, filepath.Join(dst, "entry"))
	assert.NoFileExists(t, filepath.Join(dst, "entry", "old.txt"))
}

func TestSyncDir_ReplacesFileWithDirectory(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(src, "entry"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "entry", "new.txt"), []byte("new"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dst, "entry"), []byte("old file"), 0644))

	require.NoError(t, SyncDir(src, dst))
	assert.DirExists(t, filepath.Join(dst, "entry"))
	assert.FileExists(t, filepath.Join(dst, "entry", "new.txt"))
}

func TestSyncDir_CopiesSameSizeContentWithMatchingModTime(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	srcPath := filepath.Join(src, "asset.js")
	dstPath := filepath.Join(dst, "asset.js")
	require.NoError(t, os.WriteFile(srcPath, []byte("new"), 0644))
	require.NoError(t, os.WriteFile(dstPath, []byte("old"), 0644))
	srcInfo, err := os.Stat(srcPath)
	require.NoError(t, err)
	require.NoError(t, os.Chtimes(dstPath, srcInfo.ModTime(), srcInfo.ModTime()))

	require.NoError(t, SyncDir(src, dst))
	data, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, "new", string(data))
}

func TestSyncDir_PreservesEmptyDirectory(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(src, "empty"), 0755))

	require.NoError(t, SyncDir(src, dst))
	assert.DirExists(t, filepath.Join(dst, "empty"))
}
