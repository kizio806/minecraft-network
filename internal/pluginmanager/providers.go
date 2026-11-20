package pluginmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Provider resolves immutable version metadata. It never selects a floating latest version.
type Provider interface {
	Resolve(ctx context.Context, project, versionID string) (LockEntry, error)
}

type ModrinthProvider struct {
	Client  *http.Client
	BaseURL string
}

type modrinthVersion struct {
	ID            string   `json:"id"`
	ProjectID     string   `json:"project_id"`
	VersionNumber string   `json:"version_number"`
	VersionType   string   `json:"version_type"`
	DatePublished string   `json:"date_published"`
	GameVersions  []string `json:"game_versions"`
	Loaders       []string `json:"loaders"`
	Dependencies  []struct {
		VersionID      string `json:"version_id"`
		ProjectID      string `json:"project_id"`
		DependencyType string `json:"dependency_type"`
	} `json:"dependencies"`
	Files []struct {
		URL      string `json:"url"`
		Filename string `json:"filename"`
		Size     int64  `json:"size"`
		Hashes   struct {
			SHA512 string `json:"sha512"`
			SHA256 string `json:"sha256"`
		} `json:"hashes"`
	} `json:"files"`
}

func (p ModrinthProvider) Resolve(ctx context.Context, project, versionID string) (LockEntry, error) {
	base := p.BaseURL
	if base == "" {
		base = "https://api.modrinth.com/v2"
	}
	if strings.Contains(versionID, "latest") || strings.Contains(project, "latest") {
		return LockEntry{}, fmt.Errorf("floating Modrinth identifiers are forbidden")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/version/"+versionID, nil)
	if err != nil {
		return LockEntry{}, err
	}
	client := p.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return LockEntry{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return LockEntry{}, fmt.Errorf("Modrinth returned HTTP %d", resp.StatusCode)
	}
	var v modrinthVersion
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return LockEntry{}, err
	}
	if v.ID != versionID || v.ProjectID != project {
		return LockEntry{}, fmt.Errorf("Modrinth metadata identity mismatch")
	}
	if len(v.Files) == 0 {
		return LockEntry{}, fmt.Errorf("Modrinth version has no files")
	}
	f := v.Files[0]
	return LockEntry{Project: project, Version: v.VersionNumber, VersionID: v.ID, Source: "modrinth", DownloadURL: f.URL, Filename: f.Filename, SHA256: f.Hashes.SHA256, SHA512: f.Hashes.SHA512, Size: f.Size, GameVersions: v.GameVersions, Loaders: v.Loaders, ResolvedAt: v.DatePublished}, nil
}

type GitHubProvider struct {
	Client  *http.Client
	BaseURL string
}

// Resolve resolves a release asset from GitHub API using the syntax owner/repository#asset-name and an exact release tag.
func (p GitHubProvider) Resolve(ctx context.Context, project, versionID string) (LockEntry, error) {
	parts := strings.SplitN(project, "#", 2)
	if len(parts) != 2 || strings.Contains(versionID, "latest") || strings.Contains(parts[0], "..") {
		return LockEntry{}, fmt.Errorf("GitHub requires project owner/repository#asset and exact tag")
	}
	base := p.BaseURL
	if base == "" {
		base = "https://api.github.com"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/repos/"+parts[0]+"/releases/tags/"+url.PathEscape(versionID), nil)
	if err != nil {
		return LockEntry{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	client := p.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return LockEntry{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return LockEntry{}, fmt.Errorf("GitHub returned HTTP %d", resp.StatusCode)
	}
	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name   string `json:"name"`
			URL    string `json:"browser_download_url"`
			Size   int64  `json:"size"`
			Digest string `json:"digest"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return LockEntry{}, err
	}
	if release.TagName != versionID {
		return LockEntry{}, fmt.Errorf("GitHub tag identity mismatch")
	}
	for _, a := range release.Assets {
		if a.Name == parts[1] {
			entry := LockEntry{Project: project, Version: versionID, VersionID: versionID, Source: "github", DownloadURL: a.URL, Filename: a.Name, Size: a.Size}
			if strings.HasPrefix(a.Digest, "sha256:") {
				entry.SHA256 = strings.TrimPrefix(a.Digest, "sha256:")
			}
			if entry.SHA256 == "" {
				return LockEntry{}, fmt.Errorf("GitHub asset has no SHA-256 digest")
			}
			return entry, nil
		}
	}
	return LockEntry{}, fmt.Errorf("GitHub release asset %q not found", parts[1])
}
