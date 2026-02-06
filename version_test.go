package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
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

func TestGetVersionsPaginated(t *testing.T) {
	mux, srv := setupServer(t)
	mux.HandleFunc(
		"GET /repos/cluttrdev/prebuilt/releases",
		func(w http.ResponseWriter, r *http.Request) {
			page, _ := strconv.Atoi(r.URL.Query().Get("page"))
			if page == 0 {
				page = 1
			}
			per_page := 3

			releases := []map[string]string{
				{"tag_name": "v0.6.0"},
				{"tag_name": "v0.5.0"},
				{"tag_name": "v0.4.0"},
				{"tag_name": "v0.3.0"},
				{"tag_name": "v0.2.0"},
				{"tag_name": "v0.1.0"},
			}

			w.Header().Set("Content-Type", "application/json")
			if page*per_page < len(releases) {
				w.Header().Set("Link", fmt.Sprintf(`<%s/repos/cluttrdev/prebuilt/releases?page=%d>; rel="next"`, srv.URL, page+1))
			}
			_ = json.NewEncoder(w).Encode(releases[(page-1)*per_page : page*per_page])
		},
	)

	tt := struct {
		testName string // description of this test case
		// Named input parameters for target function.
		url     string
		path    string
		want    []string
		wantErr bool
	}{
		testName: "prebuilt",
		url:      srv.URL + "/repos/cluttrdev/prebuilt/releases",
		path:     "$[*].tag_name",
		want:     []string{"v0.6.0", "v0.5.0", "v0.4.0", "v0.3.0", "v0.2.0", "v0.1.0"},
		wantErr:  false,
	}

	t.Run(tt.testName, func(t *testing.T) {
		got, gotErr := GetVersions(context.Background(), http.DefaultClient, tt.url, tt.path)
		if gotErr != nil {
			if !tt.wantErr {
				t.Errorf("GetVersionsPaginated() failed: %v", gotErr)
			}
			return
		}
		if tt.wantErr {
			t.Fatal("GetVersionsPaginated() succeeded unexpectedly")
		}
		if len(got) != len(tt.want) || got[0] != tt.want[0] {
			t.Errorf("GetVersionsPaginated() = %v, want %v", got, tt.want)
		}
	})
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
			constraints: "",
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
			constraints: ">1.7.0",
			prefix:      "jq-",
			want:        "jq-1.7.1",
			wantErr:     false,
		},
		{
			versions:    []string{"0.1.0", "1.0.0-rc1"},
			constraints: "latest",
			want:        "0.1.0",
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

func TestResolveVersion(t *testing.T) {
	tests := []struct {
		testName      string
		pages         [][]string // pages of versions; nil means no server needed
		spec          string
		prefix        string
		want          string
		wantErr       bool
		wantCallCount *int // if non-nil, verify the number of API calls
	}{
		{
			testName:      "early termination on first page match",
			pages:         [][]string{{"v2.0.0", "v1.9.0"}, {"v1.8.0", "v1.7.0"}},
			spec:          "^2.0.0",
			prefix:        "v",
			want:          "v2.0.0",
			wantCallCount: ptr(1),
		},
		{
			testName: "per-page semver sorting",
			pages:    [][]string{{"v1.5.0", "v2.0.0", "v1.4.0"}}, // out of semver order
			spec:     ">=1.0.0",
			prefix:   "v",
			want:     "v2.0.0", // highest semver, not first in list
		},
		{
			testName:      "match on second page",
			pages:         [][]string{{"v2.1.0", "v2.0.0"}, {"v1.5.0", "v1.0.0"}},
			spec:          "^1.0.0",
			prefix:        "v",
			want:          "v1.5.0",
			wantCallCount: ptr(2),
		},
		{
			testName: "no matching version",
			pages:    [][]string{{"v1.0.0", "v0.9.0"}},
			spec:     "^2.0.0",
			prefix:   "v",
			wantErr:  true,
		},
		{
			testName: "empty URL returns spec as-is",
			pages:    nil, // no server needed
			spec:     "v1.2.3",
			prefix:   "v",
			want:     "v1.2.3",
		},
		{
			testName: "latest spec",
			pages:    [][]string{{"v2.0.0", "v1.0.0"}},
			spec:     "latest",
			prefix:   "v",
			want:     "v2.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			var url string
			var client *http.Client
			var callCount int

			if tt.pages != nil {
				mux, srv := setupServer(t)
				client = srv.Client()
				url = srv.URL + "/releases"

				mux.HandleFunc("GET /releases", func(w http.ResponseWriter, r *http.Request) {
					callCount++
					page, _ := strconv.Atoi(r.URL.Query().Get("page"))
					if page == 0 {
						page = 1
					}

					releases := make([]map[string]string, len(tt.pages[page-1]))
					for i, v := range tt.pages[page-1] {
						releases[i] = map[string]string{"tag_name": v}
					}

					w.Header().Set("Content-Type", "application/json")
					if page < len(tt.pages) {
						w.Header().Set("Link", fmt.Sprintf(`<%s/releases?page=%d>; rel="next"`, srv.URL, page+1))
					}
					_ = json.NewEncoder(w).Encode(releases)
				})
			} else {
				client = http.DefaultClient
				url = ""
			}

			got, gotErr := ResolveVersion(
				context.Background(),
				client,
				url,
				"$[*].tag_name",
				tt.spec,
				tt.prefix,
			)

			if gotErr != nil {
				if !tt.wantErr {
					t.Fatalf("ResolveVersion() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("ResolveVersion() succeeded unexpectedly")
			}
			if got != tt.want {
				t.Errorf("ResolveVersion() = %v, want %v", got, tt.want)
			}
			if tt.wantCallCount != nil && callCount != *tt.wantCallCount {
				t.Errorf("expected %d API calls, got %d", *tt.wantCallCount, callCount)
			}
		})
	}
}
