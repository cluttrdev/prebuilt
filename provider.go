package main

import (
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"os"
	"slices"
	"sort"
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
	client := defaultClient()
	if spec.AuthToken != "" {
		client = newAuthedClient(spec.AuthToken)
	}

	return &Provider{
		Spec:   spec,
		Client: client,
	}
}

func InitProviders(specs []ProviderSpec, tokens map[string]string) (map[string]*Provider, error) {
	// resolve auth tokens first, so extending providers can inherit them
	for i := range len(specs) {
		// try env variables first
		if _, ok := mayBeEnvVar(specs[i].AuthToken); ok {
			specs[i].AuthToken = os.Getenv(specs[i].AuthToken)
		}
		// then try tokens map, given the providers name
		if specs[i].AuthToken == "" {
			specs[i].AuthToken = tokens[specs[i].Name]
		}
	}

	// build registry
	registry, err := buildProviderRegistry(specs)
	if err != nil {
		return nil, fmt.Errorf("build provider registry: %w", err)
	}

	providers := make(map[string]*Provider, len(registry))
	for name, spec := range registry {
		providers[name] = NewProvider(spec)
	}

	return providers, nil

	// specs = append(
	// 	// --- built-in
	// 	[]ProviderSpec{
	// 		githubProviderSpec,
	// 		gitlabProviderSpec,
	// 		httpsProviderSpec,
	// 		httpProviderSpec,
	// 	},
	// 	// ---
	// 	specs...,
	// )
}

// buildProviderRegistry creates a map of providers by name.
func buildProviderRegistry(specs []ProviderSpec) (map[string]ProviderSpec, error) {
	registry := make(map[string]ProviderSpec)
	for _, spec := range specs {
		if spec.Name == "" {
			return nil, fmt.Errorf("missing provider name")
		}
		if _, exists := registry[spec.Name]; exists {
			return nil, fmt.Errorf("duplicate provider: %s", spec.Name)
		}

		registry[spec.Name] = spec
	}

	// Detect cycles
	if err := detectProviderCycles(registry); err != nil {
		return nil, err
	}

	// Validate 'extends' references
	for name, spec := range registry {
		if spec.Extends != "" {
			if _, exists := registry[spec.Extends]; !exists {
				return nil, fmt.Errorf("provider %q extends unknown provider: %s", name, spec.Extends)
			}
		}
	}

	// Sort with merge
	processed := make(map[string]struct{})

	var process func(name string)
	process = func(name string) {
		if _, ok := processed[name]; ok {
			return
		}
		spec := registry[name]

		// Process parent first if extending
		if spec.Extends != "" {
			process(spec.Extends)
			spec = mergeProviderSpecs(registry[spec.Extends], spec)
		}

		registry[name] = spec
		processed[name] = struct{}{}
	}

	// Process all providers in deterministic order
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range slices.Sorted(maps.Keys(registry)) {
		if _, ok := processed[name]; !ok {
			process(name)
		}
	}

	return registry, nil
}

func detectProviderCycles(registry map[string]ProviderSpec) error {
	visited := make(map[string]bool)
	stack := make(map[string]bool)

	var detectCycle func(name string, chain []string) error
	detectCycle = func(name string, chain []string) error {
		if stack[name] {
			cycle := append(chain, name)
			return fmt.Errorf("circular dependency: %s", strings.Join(cycle, " -> "))
		}

		if visited[name] {
			return nil
		}

		spec, exists := registry[name]
		if !exists {
			return nil
		}

		visited[name] = true
		stack[name] = true

		if spec.Extends != "" {
			if err := detectCycle(spec.Extends, append(chain, name)); err != nil {
				return err
			}
		}

		stack[name] = false
		return nil
	}

	for name := range registry {
		if !visited[name] {
			if err := detectCycle(name, []string{}); err != nil {
				return err
			}
		}
	}

	return nil
}

func mergeProviderSpecs(parent, child ProviderSpec) ProviderSpec {
	spec := parent

	spec.Name = child.Name
	spec.Extends = child.Extends

	if child.VersionsURL != "" {
		spec.VersionsURL = child.VersionsURL
	}
	if child.VersionsJSONPath != "" {
		spec.VersionsJSONPath = child.VersionsJSONPath
	}
	if child.DownloadURL != "" {
		spec.DownloadURL = child.DownloadURL
	}

	if child.AuthToken != "" {
		spec.AuthToken = child.AuthToken
	}

	return spec
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

var builtinProviderSpecs = []ProviderSpec{
	githubProviderSpec,
	gitlabProviderSpec,
	httpsProviderSpec,
	httpProviderSpec,
}
