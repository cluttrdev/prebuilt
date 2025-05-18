package main

import (
	"fmt"
	"io"
	"net/http"
	_url "net/url"
	"os"
	"path/filepath"
)

// Download retrieves a binary asset from the given url and saves it in the
// given directory.
// It returns the local absolut path to the downloaded file.
func Download(url string, dir string) (string, error) {
	u, _ := _url.Parse(url)
	filename := filepath.Base(u.Path)

	file, err := os.Create(filepath.Join(dir, filename))
	if err != nil {
		return "", fmt.Errorf("create output file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("write output file: %w", err)
	}

	return file.Name(), err
}
