package main

import (
	"io"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		r       io.Reader
		cfg     *Config
		wantErr bool
	}{
		{
			// r: bytes.NewReader([]byte(`
			// 	global:
			// 		installDir: "/usr/local/bin"
			// 	binaries:
			// 	  - name: prebuilt
			// 		version: v0.1.0
			// 		downloadUrl: https://github.com/cluttrdev/prebuilt/releases/download/{{ .Version }}/prebuilt_{{ .Version }}_linux-amd64.tar.gz
			// 		extractPath: prebuilt
			// 	`)),
			cfg: &Config{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := LoadConfig(tt.r, tt.cfg)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("LoadConfig() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("LoadConfig() succeeded unexpectedly")
			}
		})
	}
}

func TestLoadConfigFile(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		filename string
		cfg      *Config
		wantErr  bool
	}{
		{
			filename: "testdata/config.yaml",
			cfg: &Config{},
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := LoadConfigFile(tt.filename, tt.cfg)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("LoadConfigFile() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("LoadConfigFile() succeeded unexpectedly")
			}
		})
	}
}

func Test_renderTemplate(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		tmpl    string
		bin     Binary
		want    string
		wantErr bool
	}{
		{
			tmpl: "https://github.com/cluttrdev/prebuilt/releases/download/{{ .Version }}/prebuilt_{{ .Version }}_linux-amd64.tar.gz",
			bin:  Binary{Name: "prebuilt", Version: "v0.1.0", ExtractPath: "prebuilt"},
			want: "https://github.com/cluttrdev/prebuilt/releases/download/v0.1.0/prebuilt_v0.1.0_linux-amd64.tar.gz",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := renderTemplate(tt.tmpl, tt.bin)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("renderTemplate() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("renderTemplate() succeeded unexpectedly")
			}
			if got != tt.want {
				t.Errorf("renderTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}
