package main

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_resolveProviderData(t *testing.T) {
	tests := []struct {
		testName string // description of this test case
		// Named input parameters for target function.
		s       string
		want    ProviderData
		wantErr bool
	}{
		{
			testName: "github",
			s:        "github://cluttrdev/prebuilt?asset=prebuilt_{{ .Version }}_linux-amd64.tar.gz",
			want: ProviderData{
				Scheme: "github",
				Host:   "cluttrdev",
				Path:   "prebuilt",
				Values: map[string]string{
					"asset": "prebuilt_{{ .Version }}_linux-amd64.tar.gz",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			got, gotErr := parseDSN(tt.s)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("resolveProviderData() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("resolveProviderData() succeeded unexpectedly")
			}
			if d := cmp.Diff(tt.want, got); d != "" {
				t.Errorf("resolveProviderData() mismatch (-want/+got): %v", d)
			}
		})
	}
}

func Test_buildProviderRegistry(t *testing.T) {
	builtinSpec := ProviderSpec{
		Name:             "builtin",
		VersionsURL:      "https://versions.example.com",
		VersionsJSONPath: "$[*].tag_name",
		DownloadURL:      "https://packages.example.com",
		AuthToken:        "${PREBUILT_TOKEN}",
	}
	tests := []struct {
		testName string

		specs   []ProviderSpec
		want    map[string]ProviderSpec
		wantErr string
	}{
		{
			testName: "circular dependency",
			specs: []ProviderSpec{
				{Name: "a", Extends: "b"},
				{Name: "b", Extends: "a"},
			},
			wantErr: "circular dependency",
		},
		{
			testName: "simple extend",
			specs: []ProviderSpec{
				builtinSpec,
				{
					Name:        "builtin-extended",
					Extends:     "builtin",
					DownloadURL: "https://get.extended.com",
				},
			},
			want: map[string]ProviderSpec{
				"builtin": builtinSpec,
				"builtin-extended": {
					Name:             "builtin-extended",
					VersionsURL:      builtinSpec.VersionsURL,      // inherited
					VersionsJSONPath: builtinSpec.VersionsJSONPath, // inherited
					DownloadURL:      "https://get.extended.com",   // overridden
					AuthToken:        builtinSpec.AuthToken,        // inherited

					Extends: "builtin",
				},
			},
		},
		{
			testName: "nested extend",
			specs: []ProviderSpec{
				builtinSpec,
				{
					Name:        "builtin-extended",
					Extends:     "builtin",
					DownloadURL: "https://get.extended.com",
				},
				{
					Name:      "builtin-extended-again",
					Extends:   "builtin-extended",
					AuthToken: "secret",
				},
			},
			want: map[string]ProviderSpec{
				"builtin": builtinSpec,
				"builtin-extended": {
					Name:             "builtin-extended",
					VersionsURL:      builtinSpec.VersionsURL,      // inherited
					VersionsJSONPath: builtinSpec.VersionsJSONPath, // inherited
					DownloadURL:      "https://get.extended.com",   // overridden
					AuthToken:        builtinSpec.AuthToken,        // inherited

					Extends: "builtin",
				},
				"builtin-extended-again": {
					Name:             "builtin-extended-again",
					VersionsURL:      builtinSpec.VersionsURL,      // inherited
					VersionsJSONPath: builtinSpec.VersionsJSONPath, // inherited
					DownloadURL:      "https://get.extended.com",   // overridden
					AuthToken:        "secret",                     // overridden

					Extends: "builtin-extended",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			got, err := buildProviderRegistry(tt.specs)
			if (err != nil) != (tt.wantErr != "") {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr != "" {
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErr)
				}
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("got %d providers, want %d", len(got), len(tt.want))
			}

			wantSpecs := make([]ProviderSpec, 0, len(tt.want))
			for _, name := range slices.Sorted(maps.Keys(tt.want)) {
				wantSpecs = append(wantSpecs, tt.want[name])
			}
			gotSpecs := make([]ProviderSpec, 0, len(got))
			for _, name := range slices.Sorted(maps.Keys(got)) {
				gotSpecs = append(gotSpecs, got[name])
			}

			if diff := cmp.Diff(wantSpecs, gotSpecs); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
