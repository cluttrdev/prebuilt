package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupServer(t *testing.T) (*http.ServeMux, *httptest.Server) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return mux, srv
}

func TestGetVersions(t *testing.T) {
	mux, srv := setupServer(t)
	mux.HandleFunc(
		"GET /repos/cluttrdev/prebuilt/releases",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]string{
				{
					"tag_name": "v0.1.0",
				},
			})
		},
	)

	tests := []struct {
		testName string // description of this test case
		// Named input parameters for target function.
		url     string
		path    string
		want    []string
		wantErr bool
	}{
		{
			testName: "prebuilt",
			url:      srv.URL + "/repos/cluttrdev/prebuilt/releases",
			path:     "$[*].tag_name",
			want:     []string{"v0.1.0"},
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			got, gotErr := GetVersions(context.Background(), http.DefaultClient, tt.url, tt.path)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("GetVersions() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("GetVersions() succeeded unexpectedly")
			}
			if len(got) != len(tt.want) || got[0] != tt.want[0] {
				t.Errorf("GetVersions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindLatestVersion(t *testing.T) {
	tests := []struct {
		testName string // description of this test case
		// Named input parameters for target function.
		versions    []string
		constraints string
		prefix      string
		want        string
		wantErr     bool
	}{
		{
			versions:    []string{"v0.1.0", "v0.0.1"},
			constraints: "*",
			want:        "v0.1.0",
			wantErr:     false,
		},
		{
			versions:    []string{"jq-1.7.1"},
			constraints: "*",
			prefix:      "jq-",
			want:        "jq-1.7.1",
			wantErr:     false,
		},
		{
			versions:    []string{"jq-1.7.1"},
			constraints: ">1.7",
			prefix:      "jq-",
			want:        "jq-1.7.1",
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			got, gotErr := FindLatestVersion(tt.versions, tt.constraints, tt.prefix)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("FindLatestVersion() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("FindLatestVersion() succeeded unexpectedly")
			}
			// TODO: update the condition below to compare got with tt.want.
			if got != tt.want {
				t.Errorf("FindLatestVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
