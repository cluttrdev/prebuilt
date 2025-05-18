package main

import (
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func urlMustParse(s string) url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return *u
}

func Test_resolveGitHubProvider(t *testing.T) {
	tests := []struct {
		testName string // description of this test case
		// Named input parameters for target function.
		u       url.URL
		want    ProviderSpec
		wantErr bool
	}{
		{
			u: urlMustParse("github://cluttrdev/prebuilt?asset=prebuilt_{{ .Version }}_linux-amd64.tar.gz"),
			want: ProviderSpec{
				Name:             "github",
				VersionsURL:      "https://api.github.com/repos/cluttrdev/prebuilt/releases",
				VersionsJSONPath: "$[*].tag_name",
				DownloadURL:      "https://github.com/cluttrdev/prebuilt/releases/download/{{ .Version }}/prebuilt_{{ .Version }}_linux-amd64.tar.gz",
			},
		},
		{
			u: urlMustParse("https://github.com/cluttrdev/prebuilt/releases/download/{{ .Version }}/prebuilt_{{ .Version }}_linux-amd64.tar.gz"),
			want: ProviderSpec{
				Name:             "github",
				VersionsURL:      "https://api.github.com/repos/cluttrdev/prebuilt/releases",
				VersionsJSONPath: "$[*].tag_name",
				DownloadURL:      "https://github.com/cluttrdev/prebuilt/releases/download/{{ .Version }}/prebuilt_{{ .Version }}_linux-amd64.tar.gz",
			},
		},
		{
			u:       urlMustParse("github://cluttrdev/prebuilt"),
			wantErr: true, // missing asset parameter
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			got, gotErr := resolveGitHubProvider(tt.u)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("resolveGitHubProvider() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("resolveGitHubProvider() succeeded unexpectedly")
			}

			if d := cmp.Diff(tt.want, got); d != "" {
				t.Errorf("resolveGitHubProvider() mismatch (-want/+got): %v", d)
			}
		})
	}
}
