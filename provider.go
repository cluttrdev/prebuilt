package main

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

func resolveProviderSpec(p Provider) (ProviderSpec, error) {
	switch {
	case p.String != nil:
		return resolveProviderString(*p.String)
	case p.Spec != nil:
		return *p.Spec, nil
	}
	return ProviderSpec{}, fmt.Errorf("invalid provider config")
}

func resolveProviderString(spec string) (ProviderSpec, error) {
	u, err := url.Parse(spec)
	if err != nil {
		return ProviderSpec{}, err
	}

	if u.Scheme == "github" || u.Host == "github.com" {
		return resolveGitHubProvider(*u)
	}

	return ProviderSpec{
		Name:        u.Hostname(),
		DownloadURL: spec,
	}, nil
}

func resolveGitHubProvider(u url.URL) (ProviderSpec, error) {
	const (
		githubVersionsURL      = "https://api.github.com/repos/%s/%s/releases"
		githubVersionsJSONPath = "$[*].tag_name"
		githubDownloadURL      = "https://github.com/%s/%s/releases/download/{{ .Version }}/%s"
	)

	var (
		owner string
		repo  string
		asset string
	)
	if u.Scheme == "github" {
		// github://cluttrdev/prebuilt?asset_path=prebuilt_{{ .Version }}_linux-amd64.tar.gz
		owner = u.Host
		repo = strings.TrimPrefix(u.Path, "/")
		asset = u.Query().Get("asset")
		if asset == "" {
			return ProviderSpec{}, fmt.Errorf("missing query parameter: asset")
		}
	} else if u.Host == "github.com" {
		// https://github.com/$owner/$repo/releases/download/$version/$asset_path
		parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
		owner = parts[0]
		repo = parts[1]
		asset = filepath.Join(parts[5:]...)
	} else {
		return ProviderSpec{}, fmt.Errorf("invalid url")
	}

	return ProviderSpec{
		Name:             "github",
		VersionsURL:      fmt.Sprintf(githubVersionsURL, owner, repo),
		VersionsJSONPath: githubVersionsJSONPath,
		DownloadURL:      fmt.Sprintf(githubDownloadURL, owner, repo, asset),
	}, nil
}
