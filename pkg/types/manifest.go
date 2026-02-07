package types

import "time"

// BundleManifest describes the contents of the generated archive.
type BundleManifest struct {
	// Version is the schema version of the bundle layout.
	Version string `json:"version"`

	// GeneratedAt is the timestamp when the bundle was created.
	GeneratedAt time.Time `json:"generatedAt"`

	// TotalFiles is the count of files included in the archive.
	TotalFiles int `json:"totalFiles"`

	// Files lists all files in the archive with their metadata.
	Files []FileEntry `json:"files"`

	// ContentHash is the SHA256 checksum of the entire archive content (excluding the archive wrapper itself).
	// Note: In practice, this might refer to a hash of the manifest or a deterministic hash of the file contents.
	ContentHash string `json:"contentHash"`
}

// FileEntry represents a single file inside the bundle.
type FileEntry struct {
	// Path is the relative path of the file inside the archive.
	Path string `json:"path"`

	// Size is the size of the file in bytes.
	Size int64 `json:"size"`

	// SHA256 is the checksum of the file content.
	SHA256 string `json:"sha256"`
}
