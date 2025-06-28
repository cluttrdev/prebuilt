package main

import (
	"time"
)

type BinaryData struct {
	Name        string `yaml:"name"`
	Provider    string `yaml:"provider,omitempty"`
	Version     string `yaml:"version"`
	DownloadURL string `yaml:"downloadURL"`
	ExtractPath string `yaml:"extractPath,omitempty"`
}

type Lock struct {
	Generated time.Time    `yaml:"generated"`
	Digest    string       `yaml:"digest"`
	Binaries  []BinaryData `yaml:"binaries"`
}
