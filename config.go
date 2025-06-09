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
	Global    GLobal         `yaml:"global"`
	Binaries  []BinarySpec   `yaml:"binaries"`
	Providers []ProviderSpec `yaml:"providers"`
}

// Global holds configuration settings that apply to all managed binaries.
type GLobal struct {
	InstallDir string `yaml:"installDir"`
}

// BinarySpec holds the configuration settings for a specific binary.
type BinarySpec struct {
	Name        string         `yaml:"name"`
	BinName     string         `yaml:"binName"`
	Version     Version        `yaml:"version"`
	Provider    ProviderConfig `yaml:"provider"`
	ExtractPath string         `yaml:"extractPath"`
}

type Version struct {
	String *string
	Spec   *VersionSpec
}

type VersionSpec struct {
	Prefix      string `yaml:"prefix"`
	Constraints string `yaml:"constraints"`
}

type ProviderConfig struct {
	DSN  *string
	Spec *ProviderSpec
}

type ProviderSpec struct {
	Name             string `yaml:"name"`
	VersionsURL      string `yaml:"versionsUrl"`
	VersionsJSONPath string `yaml:"versionsJsonPath"`
	DownloadURL      string `yaml:"downloadUrl"`
	AuthToken        string `yaml:"authToken"`
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
	yaml.RegisterCustomUnmarshaler(func(t *ProviderConfig, b []byte) error {
		var (
			v   any
			err error
		)
		if err = yaml.Unmarshal(b, &v); err != nil {
			return err
		}
		switch vv := v.(type) {
		case string:
			t.DSN = &vv
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

func renderTemplate(tpl string, data map[string]any) (string, error) {
	t := template.New("")
	initFuncMap(t)

	t, err := t.Parse(tpl)
	if err != nil {
		return "", err
	}

	var w bytes.Buffer
	if err := t.Execute(&w, data); err != nil {
		return "", err
	}

	return w.String(), nil
}

func initFuncMap(t *template.Template) {
	funcMap := make(template.FuncMap)

	funcMap["trimPrefix"] = func(prefix string, s string) string {
		return strings.TrimPrefix(s, prefix)
	}

	funcMap["tpl"] = func(tpl string, vals map[string]any) (string, error) {
		tt, err := t.Clone()
		if err != nil {
			return "", fmt.Errorf("clone template: %w", err)
		}
		tt, err = tt.New(t.Name()).Parse(tpl)
		if err != nil {
			return "", fmt.Errorf("parse template %q: %w", tpl, err)
		}
		var buf strings.Builder
		if err := tt.Execute(&buf, vals); err != nil {
			return "", fmt.Errorf("execute template %q: %w", tpl, err)
		}
		return buf.String(), nil
	}

	t.Funcs(funcMap)
}
