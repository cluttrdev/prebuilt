package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	_url "net/url"
	"os"
	"path/filepath"

	"go.cluttr.dev/prebuilt/internal/metaerr"
)

// Download retrieves a binary asset from the given url and saves it in the
// given directory.
// It returns the local absolut path to the downloaded file.
func Download(ctx context.Context, client *http.Client, url string, dir string) (string, error) {
	u, _ := _url.Parse(url)
	filename := filepath.Base(u.Path)

	file, err := os.Create(filepath.Join(dir, filename))
	if err != nil {
		return "", fmt.Errorf("create output file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", metaerr.WithMetadata(
			fmt.Errorf("%d - %s", resp.StatusCode, http.StatusText(resp.StatusCode)),
			"body", string(body),
		)
	}

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("write output file: %w", err)
	}

	return file.Name(), err
}
