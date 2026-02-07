package bundler

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/mrhapile/fluid-diagnose-bundler/pkg/types"
)

type ManifestBuilder struct {
	manifest types.BundleManifest
}

func NewManifestBuilder(version string, ts time.Time) *ManifestBuilder {
	return &ManifestBuilder{
		manifest: types.BundleManifest{
			Version:     version,
			GeneratedAt: ts,
			Files:       []types.FileEntry{},
		},
	}
}

func (mb *ManifestBuilder) AddFile(path string, size int64, data []byte) {
	hash := sha256.Sum256(data)
	entry := types.FileEntry{
		Path:   path,
		Size:   size,
		SHA256: hex.EncodeToString(hash[:]),
	}
	mb.manifest.Files = append(mb.manifest.Files, entry)
	mb.manifest.TotalFiles++
}

func (mb *ManifestBuilder) Build() types.BundleManifest {
	// Compute global content hash if needed (e.g. hash of concatenated file hashes)
	hasher := sha256.New()
	for _, f := range mb.manifest.Files {
		hasher.Write([]byte(f.SHA256))
	}
	mb.manifest.ContentHash = hex.EncodeToString(hasher.Sum(nil))
	return mb.manifest
}
