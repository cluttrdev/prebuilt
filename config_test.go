package main

import (
	"bytes"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func ptr[T any](v T) *T {
	return &v
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		testName string // description of this test case
		// Named input parameters for target function.
		r       io.Reader
		wantCfg Config
		wantErr bool
	}{
		{
			r: bytes.NewReader([]byte(`
global:
  installDir: "/usr/local/bin"
binaries:
  - name: prebuilt
    version: latest
    provider: github://cluttrdev/prebuilt?asset=prebuilt_{{ .Version }}_linux-amd64.tar.gz
    extractPath: prebuilt
  - name: jq
    version:
      constraints: jq-1.7.1
      prefix: "jq-"
    provider: https://github.com/jqlang/jq/releases/download/{{ .Version }}/jq-linux-amd64
  - name: helm
    version: ">=3,<4"
    provider:
      versionsUrl: https://api.github.com/repos/helm/helm/releases
      versionsJsonPath: $[*].tag_name
      downloadUrl: https://get.helm.sh/helm-{{ .Version }}-linux-amd64.tar.gz
`)),
			wantCfg: Config{
				Global: GLobal{
					InstallDir: "/usr/local/bin",
				},
				Binaries: []BinarySpec{
					{
						Name:    "prebuilt",
						Version: Version{String: ptr("latest")},
						Provider: ProviderConfig{
							DSN: ptr("github://cluttrdev/prebuilt?asset=prebuilt_{{ .Version }}_linux-amd64.tar.gz"),
						},
						ExtractPath: "prebuilt",
					},
					{
						Name:    "jq",
						Version: Version{Spec: &VersionSpec{Constraints: "jq-1.7.1", Prefix: "jq-"}},
						Provider: ProviderConfig{
							DSN: ptr("https://github.com/jqlang/jq/releases/download/{{ .Version }}/jq-linux-amd64"),
						},
					},
					{
						Name:    "helm",
						Version: Version{String: ptr(">=3,<4")},
						Provider: ProviderConfig{
							Spec: &ProviderSpec{
								VersionsURL:      "https://api.github.com/repos/helm/helm/releases",
								VersionsJSONPath: "$[*].tag_name",
								DownloadURL:      "https://get.helm.sh/helm-{{ .Version }}-linux-amd64.tar.gz",
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			var gotCfg Config
			gotErr := LoadConfig(tt.r, &gotCfg)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("LoadConfig() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("LoadConfig() succeeded unexpectedly")
			}

			if d := cmp.Diff(tt.wantCfg, gotCfg); d != "" {
				t.Errorf("LoadConfig() mismatch (- want, + got): %s", d)
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
			cfg:      &Config{},
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
		data    map[string]any
		want    string
		wantErr bool
	}{
		{
			tmpl: "https://github.com/cluttrdev/prebuilt/releases/download/{{ .Version }}/prebuilt_{{ .Version }}_linux-amd64.tar.gz",
			data: map[string]any{"Version": "v0.1.0"},
			want: "https://github.com/cluttrdev/prebuilt/releases/download/v0.1.0/prebuilt_v0.1.0_linux-amd64.tar.gz",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := renderTemplate(tt.tmpl, tt.data)
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

// func Test_resolveBinarySpec(t *testing.T) {
// 	mux, srv := setupServer(t)
//
// 	mux.HandleFunc(
// 		"GET /versions",
// 		func(w http.ResponseWriter, r *http.Request) {
// 			w.Header().Set("Content-Type", "application/json")
// 			_ = json.NewEncoder(w).Encode([]map[string]string{
// 				{"tag_name": "v1.0.0"},
// 				{"tag_name": "v0.1.0"},
// 				{"tag_name": "v0.0.1"},
// 			})
// 		},
// 	)
//
// 	providerSpec := &ProviderSpec{
// 		VersionsURL:      srv.URL + "/versions",
// 		VersionsJSONPath: "$[*].tag_name",
// 		DownloadURL:      srv.URL + "/download/{{ .Version }}/asset",
// 	}
// 	latestData := binaryData{
// 		Name:        "prebuilt",
// 		Version:     "v1.0.0",
// 		DownloadURL: srv.URL + "/download/v1.0.0/asset",
// 	}
//
// 	tests := []struct {
// 		testName string // description of this test case
// 		// Named input parameters for target function.
// 		bin     BinarySpec
// 		want    binaryData
// 		wantErr bool
// 	}{
// 		{
// 			testName: "latestVersion",
// 			bin: BinarySpec{
// 				Name:     "prebuilt",
// 				Version:  Version{String: ptr("v1.0.0")},
// 				Provider: Provider{Spec: providerSpec},
// 			},
// 			want:    latestData,
// 			wantErr: false,
// 		},
// 		{
// 			testName: "latestString",
// 			bin: BinarySpec{
// 				Name:     "prebuilt",
// 				Version:  Version{String: ptr("latest")},
// 				Provider: Provider{Spec: providerSpec},
// 			},
// 			want:    latestData,
// 			wantErr: false,
// 		},
// 		{
// 			testName: "latestStar",
// 			bin: BinarySpec{
// 				Name:     "prebuilt",
// 				Version:  Version{String: ptr("*")},
// 				Provider: Provider{Spec: providerSpec},
// 			},
// 			want:    latestData,
// 			wantErr: false,
// 		},
// 		{
// 			testName: "latestEmpty",
// 			bin: BinarySpec{
// 				Name:     "prebuilt",
// 				Version:  Version{String: ptr("")},
// 				Provider: Provider{Spec: providerSpec},
// 			},
// 			want:    latestData,
// 			wantErr: false,
// 		},
// 		{
// 			bin: BinarySpec{
// 				Name:     "prebuilt",
// 				Version:  Version{String: ptr("v0.1.0")},
// 				Provider: Provider{Spec: providerSpec},
// 			},
// 			want: binaryData{
// 				Name:        "prebuilt",
// 				Version:     "v0.1.0",
// 				DownloadURL: srv.URL + "/download/v0.1.0/asset",
// 			},
// 			wantErr: false,
// 		},
// 		{
// 			bin: BinarySpec{
// 				Name:     "prebuilt",
// 				Version:  Version{String: ptr("<1.0")},
// 				Provider: Provider{Spec: providerSpec},
// 			},
// 			want: binaryData{
// 				Name:        "prebuilt",
// 				Version:     "v0.1.0",
// 				DownloadURL: srv.URL + "/download/v0.1.0/asset",
// 			},
// 			wantErr: false,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.testName, func(t *testing.T) {
// 			got, gotErr := resolveBinarySpec(tt.bin)
// 			if gotErr != nil {
// 				if !tt.wantErr {
// 					t.Errorf("resolveBinarySpec() failed: %v", gotErr)
// 				}
// 				return
// 			}
// 			if tt.wantErr {
// 				t.Fatal("resolveBinarySpec() succeeded unexpectedly")
// 			}
// 			if d := cmp.Diff(tt.want, got); d != "" {
// 				t.Errorf("resolveBinarySpec() mismatch (-want/+got): %s", d)
// 			}
// 		})
// 	}
// }
