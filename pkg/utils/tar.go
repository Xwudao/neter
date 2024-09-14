package utils

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Tar creates a tarball from the source directories/files and writes it to the target file.
func Tar(sources []string, target string) error {
	// Create a new tar file
	tarfile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer tarfile.Close()

	// Create a gzip.Writer
	gzipWriter := gzip.NewWriter(tarfile)
	defer gzipWriter.Close()

	// Create a tar.Writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for _, source := range sources {
		source = filepath.Clean(source)
		dir := filepath.Base(source)

		// Walk through the source directory and write files to the tar file
		err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Get file info
			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}

			// Rewrite header.Name field
			header.Name = filepath.ToSlash(filepath.Join(dir, strings.TrimPrefix(path, source)))
			if info.IsDir() {
				header.Name += "/"
			}
			// Write file info
			if err := tarWriter.WriteHeader(header); err != nil {
				return err
			}

			// If it's a file, write file content
			if !info.IsDir() {
				file, err := os.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()

				if _, err := io.Copy(tarWriter, file); err != nil {
					return err
				}
			}

			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}
