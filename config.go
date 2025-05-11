package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/goccy/go-yaml"
)

// Config holds all applications configuration settings.
type Config struct {
	Global   GLobal   `yaml:"global"`
	Binaries []Binary `yaml:"binaries"`
}

// Global holds configuration settings that apply to all managed binaries.
type GLobal struct {
	InstallDir string `yaml:"installDir"`
}

// Binary holds the configuration settings for a specific binary.
type Binary struct {
	Name        string `yaml:"name"`
	BinName     string `yaml:"binName"`
	Version     string `yaml:"version"`
	DownloadURL string `yaml:"downloadUrl"`
	ExtractPath string `yaml:"extractPath"`
}

// LoadConfig reads the applications configuration from the given reader into `cfg`.
func LoadConfig(r io.Reader, cfg *Config) error {
	return yaml.NewDecoder(r).Decode(cfg)
}

// LoadConfigFile reads the applications configuration from the given file into `cfg`.
func LoadConfigFile(name string, cfg *Config) error {
	file, err := os.Open(name)
	if err != nil {
		return err
	}
	return LoadConfig(file, cfg)
}

func renderTemplate(tmpl string, bin Binary) (string, error) {
	tpl := template.New(bin.Name)

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
	if err := tpl.Execute(&w, bin); err != nil {
		return "", err
	}

	return w.String(), nil
}
