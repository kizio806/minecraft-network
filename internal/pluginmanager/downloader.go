package pluginmanager

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxArtifactSize int64 = 100 << 20

var trustedHosts = map[string]bool{
	"cdn.modrinth.com":                     true,
	"api.modrinth.com":                     true,
	"github.com":                           true,
	"api.github.com":                       true,
	"release-assets.githubusercontent.com": true,
	"repo.auxilor.io":                      true,
}

// DefaultHTTPClient is the default HTTP client used for downloading plugins and components.
var DefaultHTTPClient = &http.Client{
	Timeout: 2 * time.Minute,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	},
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return errors.New("too many redirects")
		}
		if req.URL.Scheme != "https" || !trustedHosts[strings.ToLower(req.URL.Hostname())] {
			return errors.New("redirect leaves trusted HTTPS hosts")
		}
		return nil
	},
}

func download(ctx context.Context, entry LockEntry, destination string, customClient ...*http.Client) error {
	client := DefaultHTTPClient
	if len(customClient) > 0 && customClient[0] != nil {
		client = customClient[0]
	}
	u, err := url.Parse(entry.DownloadURL)
	if err != nil || u.Scheme != "https" || !trustedHosts[strings.ToLower(u.Hostname())] {
		return errors.New("download URL is not HTTPS or trusted")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, entry.DownloadURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > maxArtifactSize {
		return errors.New("artifact exceeds maximum size")
	}
	f, err := os.CreateTemp(filepath.Dir(destination), ".artifact-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	limitReader := io.LimitReader(resp.Body, maxArtifactSize+1)
	n, err := io.Copy(f, limitReader)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if n > maxArtifactSize {
		return errors.New("artifact exceeds maximum size")
	}
	if err = verifyFile(tmp, entry); err != nil {
		return fmt.Errorf("download integrity: %w", err)
	}
	return os.Rename(tmp, destination)
}
