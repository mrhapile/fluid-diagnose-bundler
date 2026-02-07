package bundler

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// ArchiveWriter handles the creation of the .tar.gz file with deterministic ordering.
type ArchiveWriter struct {
	baseDir string // The root directory name inside the archive (e.g. fluid-diagnose-xyz)
	files   map[string][]byte
	ts      time.Time
}

// NewArchiveWriter creates a new writer instance.
func NewArchiveWriter(baseDir string, ts time.Time) *ArchiveWriter {
	return &ArchiveWriter{
		baseDir: baseDir,
		files:   make(map[string][]byte),
		ts:      ts,
	}
}

// AddFile adds a file to be included in the archive.
// path should be relative to the archive root (e.g. "manifest.json", "logs/foo.log").
func (w *ArchiveWriter) AddFile(path string, content []byte) {
	w.files[path] = content
}

// WriteToDisk creates the .tar.gz file at the specified output directory.
// It returns the absolute path to the created file and the total uncompressed size.
func (w *ArchiveWriter) WriteToDisk(outputDir string) (string, int64, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", 0, fmt.Errorf("failed to create output directory: %w", err)
	}

	filename := fmt.Sprintf("%s.tar.gz", w.baseDir)
	archivePath := filepath.Join(outputDir, filename)
	absPath, err := filepath.Abs(archivePath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	f, err := os.Create(absPath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create archive file: %w", err)
	}
	defer f.Close()

	// Use gzip compression
	gw := gzip.NewWriter(f)
	defer gw.Close()

	// Use tar writer
	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Sort files by path for deterministic output
	paths := make([]string, 0, len(w.files))
	for p := range w.files {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var totalSize int64
	for _, p := range paths {
		content := w.files[p]
		fullPath := filepath.Join(w.baseDir, p)

		header := &tar.Header{
			Name:       fullPath,
			Size:       int64(len(content)),
			Mode:       0644,
			ModTime:    w.ts,
			AccessTime: w.ts,
			ChangeTime: w.ts,
			Typeflag:   tar.TypeReg,
		}

		if err := tw.WriteHeader(header); err != nil {
			return "", 0, fmt.Errorf("failed to write header for %s: %w", p, err)
		}

		if _, err := tw.Write(content); err != nil {
			return "", 0, fmt.Errorf("failed to write content for %s: %w", p, err)
		}
		totalSize += int64(len(content))
	}

	return absPath, totalSize, nil
}
