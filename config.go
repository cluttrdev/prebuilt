package main

import (
	"bytes"
	"fmt"
	"io"
	_url "net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"

	"github.com/goccy/go-yaml"
)

// Config holds all applications configuration settings.
type Config struct {
	Global   GLobal       `yaml:"global"`
	Binaries []BinarySpec `yaml:"binaries"`
}

// Global holds configuration settings that apply to all managed binaries.
type GLobal struct {
	InstallDir string `yaml:"installDir"`
}

// BinarySpec holds the configuration settings for a specific binary.
type BinarySpec struct {
	Name        string   `yaml:"name"`
	BinName     string   `yaml:"binName"`
	Version     Version  `yaml:"version"`
	Provider    Provider `yaml:"provider"`
	ExtractPath string   `yaml:"extractPath"`
}

type Version struct {
	String *string
	Spec   *VersionSpec
}

type VersionSpec struct {
	Prefix      string `yaml:"prefix"`
	Constraints string `yaml:"constraints"`
}

type Provider struct {
	String *string
	Spec   *ProviderSpec
}

type ProviderSpec struct {
	Name             string `yaml:"name"`
	VersionsURL      string `yaml:"versionsUrl"`
	VersionsJSONPath string `yaml:"versionsJsonPath"`
	DownloadURL      string `yaml:"downloadUrl"`
}

type binaryData struct {
	Name        string
	Version     string
	DownloadURL string
	ExtractPath string
}

// LoadConfig reads the configuration from a reader into `cfg`.
func LoadConfig(r io.Reader, cfg *Config) error {
	if r == nil {
		return nil
	}
	yaml.RegisterCustomUnmarshaler(func(t *Version, b []byte) error {
		var (
			v   any
			err error
		)
		if err = yaml.Unmarshal(b, &v); err != nil {
			return err
		}
		switch vv := v.(type) {
		case string:
			t.String = &vv
		case map[string]any:
			var tt VersionSpec
			if err = yaml.Unmarshal(b, &tt); err == nil {
				t.Spec = &tt
			}
		default:
			err = fmt.Errorf("invalid type: %v", reflect.TypeOf(v))
		}
		return err
	})
	yaml.RegisterCustomUnmarshaler(func(t *Provider, b []byte) error {
		var (
			v   any
			err error
		)
		if err = yaml.Unmarshal(b, &v); err != nil {
			return err
		}
		switch vv := v.(type) {
		case string:
			t.String = &vv
		case map[string]any:
			var tt ProviderSpec
			if err = yaml.Unmarshal(b, &tt); err == nil {
				t.Spec = &tt
			}
		default:
			err = fmt.Errorf("invalid type: %v", reflect.TypeOf(v))
		}
		return err
	})
	return yaml.NewDecoder(r).Decode(cfg)
}

// LoadConfigFile reads the configuration a file into `cfg`.
func LoadConfigFile(name string, cfg *Config) error {
	file, err := os.Open(name)
	if err != nil {
		return err
	}
	return LoadConfig(file, cfg)
}

func resolveBinarySpec(bin BinarySpec) (binaryData, error) {
	var (
		data binaryData
		err  error
	)

	// Provider
	providerSpec, err := resolveProviderSpec(bin.Provider)
	if err != nil {
		return data, err
	}

	var versionSpec VersionSpec
	if bin.Version.String != nil {
		versionSpec.Constraints = *bin.Version.String
	} else if bin.Version.Spec != nil {
		versionSpec = *bin.Version.Spec
	}

	// Name
	data.Name = getBinName(bin, providerSpec)

	// Version
	data.Version, err = ResolveVersion(providerSpec.VersionsURL, providerSpec.VersionsJSONPath, versionSpec.Constraints, versionSpec.Prefix)
	if err != nil {
		return data, fmt.Errorf("resolve version: %w", err)
	}

	// DownloadURL
	data.DownloadURL, err = renderTemplate(providerSpec.DownloadURL, tplData{Version: data.Version})
	if err != nil {
		return data, fmt.Errorf("render download url: %w", err)
	}

	// ExtractPath
	data.ExtractPath, err = renderTemplate(bin.ExtractPath, tplData{Version: data.Version})
	if err != nil {
		return data, fmt.Errorf("render extract path: %w", err)
	}

	return data, err
}

func getBinName(bin BinarySpec, provider ProviderSpec) string {
	switch {
	case bin.BinName != "":
		return bin.BinName
	case bin.Name != "":
		return bin.Name
	case bin.ExtractPath != "":
		return filepath.Base(bin.ExtractPath)
	}
	return filepath.Base(urlPath(provider.DownloadURL))
}

func urlPath(url string) string {
	u, _ := _url.Parse(url)
	return u.Path
}

type tplData struct {
	Name    string
	Version string
}

func renderTemplate(tmpl string, data tplData) (string, error) {
	tpl := template.New("")

	tpl = tpl.Funcs(template.FuncMap{
		"trimPrefix": func(prefix string, s string) string {
			return strings.TrimPrefix(s, prefix)
		},
	})

	tpl, err := tpl.Parse(tmpl)
	if err != nil {
		return "", err
	}

	var w bytes.Buffer
	if err := tpl.Execute(&w, data); err != nil {
		return "", err
	}

	return w.String(), nil
}
