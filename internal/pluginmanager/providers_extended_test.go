package pluginmanager

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestModrinthProvider_ErrorPaths(t *testing.T) {
	ctx := context.Background()

	t.Run("identity mismatch", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"id":"v1","project_id":"wrong_project","version_number":"1.0.0","files":[{"url":"https://cdn.modrinth.com/p/v.jar","filename":"v.jar","size":3,"hashes":{"sha512":"abc"}}]}`))
		}))
		defer server.Close()

		provider := ModrinthProvider{BaseURL: server.URL, Client: server.Client()}
		if _, err := provider.Resolve(ctx, "correct_project", "v1"); err == nil || !strings.Contains(err.Error(), "identity mismatch") {
			t.Fatalf("expected identity mismatch error, got: %v", err)
		}
	})

	t.Run("empty files list", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"id":"v1","project_id":"p1","version_number":"1.0.0","files":[]}`))
		}))
		defer server.Close()

		provider := ModrinthProvider{BaseURL: server.URL, Client: server.Client()}
		if _, err := provider.Resolve(ctx, "p1", "v1"); err == nil || !strings.Contains(err.Error(), "no files") {
			t.Fatalf("expected no files error, got: %v", err)
		}
	})

	t.Run("malformed json", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`not a json object`))
		}))
		defer server.Close()

		provider := ModrinthProvider{BaseURL: server.URL, Client: server.Client()}
		if _, err := provider.Resolve(ctx, "p1", "v1"); err == nil {
			t.Fatal("expected error on malformed JSON, got nil")
		}
	})
}

func TestGitHubProvider_ErrorPaths(t *testing.T) {
	ctx := context.Background()

	t.Run("tag mismatch", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"tag_name":"v2.0.0","assets":[{"name":"plugin.jar","browser_download_url":"https://github.com/o/r/releases/download/v1.0.0/plugin.jar","size":3,"digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`))
		}))
		defer server.Close()

		provider := GitHubProvider{BaseURL: server.URL, Client: server.Client()}
		if _, err := provider.Resolve(ctx, "o/r#plugin.jar", "v1.0.0"); err == nil || !strings.Contains(err.Error(), "tag identity mismatch") {
			t.Fatalf("expected tag identity mismatch error, got: %v", err)
		}
	})

	t.Run("missing digest", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"tag_name":"v1.0.0","assets":[{"name":"plugin.jar","browser_download_url":"https://github.com/o/r/releases/download/v1.0.0/plugin.jar","size":3,"digest":""}]}`))
		}))
		defer server.Close()

		provider := GitHubProvider{BaseURL: server.URL, Client: server.Client()}
		if _, err := provider.Resolve(ctx, "o/r#plugin.jar", "v1.0.0"); err == nil || !strings.Contains(err.Error(), "no SHA-256 digest") {
			t.Fatalf("expected no SHA-256 digest error, got: %v", err)
		}
	})

	t.Run("asset not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"tag_name":"v1.0.0","assets":[{"name":"other.jar","browser_download_url":"https://github.com/o/r/releases/download/v1.0.0/other.jar","size":3,"digest":"sha256:aaa"}]}`))
		}))
		defer server.Close()

		provider := GitHubProvider{BaseURL: server.URL, Client: server.Client()}
		if _, err := provider.Resolve(ctx, "o/r#plugin.jar", "v1.0.0"); err == nil || !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected asset not found error, got: %v", err)
		}
	})

	t.Run("http 404", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		}))
		defer server.Close()

		provider := GitHubProvider{BaseURL: server.URL, Client: server.Client()}
		if _, err := provider.Resolve(ctx, "o/r#plugin.jar", "v1.0.0"); err == nil || !strings.Contains(err.Error(), "HTTP 404") {
			t.Fatalf("expected HTTP 404 error, got: %v", err)
		}
	})

	t.Run("malformed json", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`invalid json`))
		}))
		defer server.Close()

		provider := GitHubProvider{BaseURL: server.URL, Client: server.Client()}
		if _, err := provider.Resolve(ctx, "o/r#plugin.jar", "v1.0.0"); err == nil {
			t.Fatal("expected error on malformed JSON, got nil")
		}
	})
}
