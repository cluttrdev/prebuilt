package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/AsaiYusuke/jsonpath"
	"github.com/hashicorp/go-version"
	"go.cluttr.dev/prebuilt/internal/metaerr"
)

// ResolveVersion returns the latest version that matches the given spec.
// It queries the `url` and filters the response with the JSONPath `path` to
// retrieve a list of available versions.
// The `spec` constraints are then used to determine the latest version.
// If no url is given, the spec is returned as-is.
func ResolveVersion(url string, path string, spec string, prefix string) (string, error) {
	if url == "" {
		return spec, nil
	}

	versions, err := GetVersions(url, path)
	if err != nil {
		return "", err
	}

	return FindLatestVersion(versions, spec, prefix)
}

// GetVersions queries the `url` and filters the response using the JSONPath
// `path` to get a list of versions.
func GetVersions(url string, path string) ([]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, metaerr.WithMetadata(
			fmt.Errorf("%d - %s", resp.StatusCode, http.StatusText(resp.StatusCode)),
			"body", string(body),
		)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	var src any
	if err := json.Unmarshal(body, &src); err != nil {
		return nil, fmt.Errorf("unmarshal response body: %w", err)
	}

	return retrieveVersions(src, path)
}

// FindLatestVersion returns the latest version from the list of `versions`
// that matches the given constraints `spec`.
func FindLatestVersion(versions []string, spec string, prefix string) (string, error) {
	var constraint *version.Constraints
	switch spec {
	case "", "*", "latest":
	default:
		c, err := version.NewConstraint(strings.TrimPrefix(spec, prefix))
		if err != nil {
			return "", err
		}
		constraint = &c
	}

	vs := make([]*version.Version, 0, len(versions))
	for _, raw := range versions {
		v, err := version.NewVersion(strings.TrimPrefix(raw, prefix))
		if err != nil {
			// return "", fmt.Errorf("parse version: %w", err)
			continue
		}
		if constraint != nil && !constraint.Check(v) {
			continue
		}
		if v.Prerelease() != "" {
			continue
		}
		vs = append(vs, v)
	}
	if len(vs) == 0 {
		return "", fmt.Errorf("no matching versions: %v", spec)
	}

	sort.Sort(sort.Reverse(version.Collection(vs)))
	latest := prefix + vs[0].Original()
	return latest, nil
}

func retrieveVersions(src any, path string) ([]string, error) {
	config := jsonpath.Config{}
	config.SetAccessorMode()

	results, err := jsonpath.Retrieve(path, src, config)
	if err != nil {
		return nil, err
	}

	var versions []string
	for _, result := range results {
		version := result.(jsonpath.Accessor).Get().(string)
		if version == "" {
			continue
		}
		versions = append(versions, version)
	}

	return versions, nil
}
