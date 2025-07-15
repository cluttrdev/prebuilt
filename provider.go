package main

import (
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
)

type ProviderData struct {
	Scheme string
	Host   string
	Path   string
	Values map[string]string
}

type Provider struct {
	Spec   ProviderSpec
	Client *http.Client
}

func NewProvider(spec ProviderSpec) *Provider {
	mayBeEnvVar := func(s string) (string, bool) {
		pattern := regexp.MustCompile(`\$\{(?<name>[a-zA-Z_]+[a-zA-Z0-9_]*)\}`)
		matches := pattern.FindStringSubmatch(s)
		if matches == nil {
			return "", false
		}
		return matches[1], true
	}

	if key, ok := mayBeEnvVar(spec.AuthToken); ok {
		if token := os.Getenv(key); token != "" {
			spec.AuthToken = token
		} else {
			spec.AuthToken = ""
		}
	}

	client := defaultClient()
	if spec.AuthToken != "" {
		client = newAuthedClient(spec.AuthToken)
	}

	return &Provider{
		Spec:   spec,
		Client: client,
	}
}

func parseDSN(dsn string) (ProviderData, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return ProviderData{}, err
	}

	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return ProviderData{}, err
	}

	values := make(map[string]string)
	for k := range q {
		values[k] = q.Get(k)
	}

	return ProviderData{
		Scheme: u.Scheme,
		Host:   u.Host,
		Path:   strings.TrimPrefix(u.Path, "/"),
		Values: values,
	}, nil
}

var githubProviderSpec = ProviderSpec{
	Name:             "github",
	VersionsURL:      "https://api.github.com/repos/{{ .Provider.Host }}/{{ .Provider.Path }}/releases?per_page=100",
	VersionsJSONPath: "$[*].tag_name",
	DownloadURL:      "https://github.com/{{ .Provider.Host }}/{{ .Provider.Path }}/releases/download/{{ .Version }}/{{ tpl .Provider.Values.asset . }}",
	AuthToken:        "${PREBUILT_GITHUB_TOKEN}",
}

var gitlabProviderSpec = ProviderSpec{
	Name:             "gitlab",
	VersionsURL:      `https://gitlab.com/api/v4/projects/{{ printf "%s/%s" .Provider.Host .Provider.Path | urlquery }}/releases?per_page=100`,
	VersionsJSONPath: "$[*].tag_name",
	DownloadURL:      "https://gitlab.com/{{ .Provider.Host }}/{{ .Provider.Path }}/-/releases/{{ .Version }}/downloads/{{ tpl .Provider.Values.asset . }}",
	AuthToken:        "${PREBUILT_GITLAB_TOKEN}",
}

var httpProviderSpec = ProviderSpec{
	Name:        "http",
	DownloadURL: "http://{{ .Provider.Host }}/{{ .Provider.Path }}",
}

var httpsProviderSpec = ProviderSpec{
	Name:        "https",
	DownloadURL: "https://{{ .Provider.Host }}/{{ .Provider.Path }}",
}
