package main

import (
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
				Name: "github",
				Host: "cluttrdev",
				Path: "prebuilt",
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
