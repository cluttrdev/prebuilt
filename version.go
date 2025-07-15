package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/AsaiYusuke/jsonpath"
	"github.com/Masterminds/semver/v3"
	"go.cluttr.dev/prebuilt/internal/metaerr"
)

// ResolveVersion returns the latest version that matches the given spec.
// It queries the `url` and filters the response with the JSONPath `path` to
// retrieve a list of available versions.
// The `spec` constraints are then used to determine the latest version.
// If no url is given, the spec is returned as-is.
func ResolveVersion(ctx context.Context, client *http.Client, url string, path string, spec string, prefix string) (string, error) {
	if url == "" {
		return spec, nil
	}

	versions, err := GetVersions(ctx, client, url, path)
	if err != nil {
		return "", err
	}

	return FindLatestVersion(versions, spec, prefix)
}

// GetVersions queries the `url` and filters the response using the JSONPath
// `path` to get a list of versions.
func GetVersions(ctx context.Context, client *http.Client, url string, path string) ([]string, error) {
	var versions []string

	for {
		resp, err := client.Get(url)
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

		vs, err := retrieveVersions(src, path)
		if err != nil {
			return nil, err
		}
		versions = append(versions, vs...)

		nextLink := findNextLink(resp.Header.Values("Link"))
		if nextLink == "" {
			break
		}
		url = nextLink
	}

	return versions, nil
}

// FindLatestVersion returns the latest version from the list of `versions`
// that matches the given constraints `spec`.
func FindLatestVersion(versions []string, spec string, prefix string) (string, error) {
	if spec == "" || spec == "latest" {
		spec = "*"
	}
	constraints, err := semver.NewConstraint(strings.TrimPrefix(spec, prefix))
	if err != nil {
		return "", err
	}

	vs := make([]*semver.Version, 0, len(versions))
	for _, raw := range versions {
		v, err := semver.NewVersion(strings.TrimPrefix(raw, prefix))
		if err != nil {
			continue
		}
		if !constraints.Check(v) {
			continue
		}
		vs = append(vs, v)
	}
	if len(vs) == 0 {
		return "", fmt.Errorf("no matching versions: %v", spec)
	}

	sort.Sort(sort.Reverse(semver.Collection(vs)))
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

func findNextLink(headers []string) string {
	for _, raw := range headers {
		// Header values may be comma delimited sequences
		for _, header := range strings.Split(raw, ",") {
			var linkURL, linkRel string

			// Link header values have the form: <url>; rel="next"; foo="bar"
			for _, part := range strings.Split(header, ";") {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}

				// <url>
				if part[0] == '<' && part[len(part)-1] == '>' {
					linkURL = strings.Trim(part, "<>")
					continue
				}

				// rel="next"
				keyval := strings.SplitN(part, "=", 2)
				if len(keyval) != 2 {
					continue
				} else if strings.ToLower(keyval[0]) == "rel" {
					linkRel = strings.Trim(keyval[1], "\"")
				}
			}

			if strings.ToLower(linkRel) == "next" {
				return linkURL
			}
		}
	}
	return ""
}
