package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Extract opens the given archive and retrieves the file specified by path.
// It returns the local absolute path to the extracted file.
func Extract(archive string, path string) (string, error) {
	in, err := os.Open(archive)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = in.Close()
	}()

	reader, err := newArchiveFileReader(in, path)
	if err != nil {
		return "", err
	}

	dst := filepath.Join(filepath.Dir(archive), path)
	if err := os.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil {
		return "", err
	}

	out, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = out.Close()
	}()

	_, err = io.Copy(out, reader)
	if err != nil {
		return "", err
	}

	return dst, nil
}

func newArchiveFileReader(archive *os.File, filename string) (io.Reader, error) {
	name := archive.Name()
	switch {
	case strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".tgz"):
		gzReader, err := gzip.NewReader(archive)
		if err != nil {
			return nil, err
		}
		tarReader := tar.NewReader(gzReader)
		for {
			header, err := tarReader.Next()
			if err != nil {
				break
			}
			if header.Name == filename {
				return tarReader, nil
			}
		}
		return nil, fmt.Errorf("file not found: %v", filename)
	case strings.HasSuffix(name, ".zip"):
		stat, err := archive.Stat()
		if err != nil {
			return nil, err
		}
		zipReader, err := zip.NewReader(archive, stat.Size())
		if err != nil {
			return nil, err
		}
		return zipReader.Open(filename)
	}

	return nil, fmt.Errorf("unsupported archive")
}
