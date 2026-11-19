package backup

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ErrUnsafeArchivePath indicates a file header in the archive attempts directory traversal.
var ErrUnsafeArchivePath = errors.New("archive entry attempts directory traversal")

// ErrArchiveEntryTooLarge indicates a single entry exceeds the maximum allowed size.
var ErrArchiveEntryTooLarge = errors.New("archive entry exceeds maximum allowed size")

const maxSingleRestoreFileSize int64 = 500 << 20 // 500 MB limit per extracted entry

// CreateBackupArchive bundles repository directories and extra staged files into a compressed tarball.
func CreateBackupArchive(root, stage, destination string) error {
	f, err := os.OpenFile(destination+".tmp", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	paths := []string{"configs", "docker", "infrastructure", "servers", "secrets", ".env"}
	for _, rel := range paths {
		if err := AddTree(tw, root, rel); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if err := AddPath(tw, filepath.Join(stage, "postgres.dump"), "database/postgres.dump"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(destination+".tmp", destination)
}

// AddPath appends a single file to a tar writer with the given archive path.
func AddPath(tw *tar.Writer, path, archiveName string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	h, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	h.Name = archiveName
	if err = tw.WriteHeader(h); err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(tw, f)
	return err
}

// AddTree recursively traverses a directory tree and writes files to a tar writer.
func AddTree(tw *tar.Writer, root, rel string) error {
	path := filepath.Join(root, rel)
	_, err := os.Stat(path)
	if err != nil {
		return err
	}
	return filepath.Walk(path, func(p string, i os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		name, _ := filepath.Rel(root, p)
		if i.IsDir() {
			return nil
		}
		if strings.Contains(name, "/logs/") || strings.Contains(name, "/cache/") {
			return nil
		}
		h, err := tar.FileInfoHeader(i, "")
		if err != nil {
			return err
		}
		h.Name = name
		if err = tw.WriteHeader(h); err != nil {
			return err
		}
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
}

// ExtractBackupArchive safely unpackages a tar.gz into target directory preventing Zip Slip.
func ExtractBackupArchive(plainArchive, tempDir string) error {
	f, err := os.Open(plainArchive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	cleanTempPrefix := filepath.Clean(tempDir) + string(filepath.Separator)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		name := filepath.Clean(h.Name)
		if filepath.IsAbs(name) || strings.HasPrefix(name, "..") || strings.Contains(name, "/../") || strings.Contains(name, `\..\`) {
			return fmt.Errorf("%w: %s", ErrUnsafeArchivePath, h.Name)
		}
		cleanDest := filepath.Clean(filepath.Join(tempDir, name))
		if !strings.HasPrefix(cleanDest, cleanTempPrefix) {
			return fmt.Errorf("%w: %s", ErrUnsafeArchivePath, h.Name)
		}
		dest := cleanDest
		if h.FileInfo().IsDir() {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
			return err
		}
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if err != nil {
			return err
		}
		limited := io.LimitReader(tr, maxSingleRestoreFileSize+1)
		n, copyErr := io.Copy(out, limited)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if n > maxSingleRestoreFileSize {
			return fmt.Errorf("%w: %s exceeds %d bytes", ErrArchiveEntryTooLarge, h.Name, maxSingleRestoreFileSize)
		}
	}
	return nil
}
