package bundler

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/mrhapile/fluid-diagnose-bundler/pkg/types"
	yaml "gopkg.in/yaml.v2"
)

// Option configures the bundling process.
type Option func(*config)

type config struct {
	redact    bool
	timestamp time.Time
	outputDir string
}

// WithRedaction enables sensitive data redaction.
func WithRedaction() Option {
	return func(c *config) {
		c.redact = true
	}
}

// WithTimestamp sets a specific timestamp for deterministic output.
// If zero, defaults to time.Now() (which breaks determinism across runs).
func WithTimestamp(t time.Time) Option {
	return func(c *config) {
		c.timestamp = t
	}
}

// WithOutputDir sets the directory where the archive will be written.
func WithOutputDir(path string) Option {
	return func(c *config) {
		c.outputDir = path
	}
}

// Build creates a diagnostic bundle from the given input.
func Build(input types.BundleInput, opts ...Option) (*types.BundleResult, error) {
	// 1. Configure
	cfg := &config{
		timestamp: time.Now(), // Default, can be overridden for determinism
		outputDir: ".",
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// 2. Initialize components
	// Determine base directory name
	// In a real scenario, this might come from input.Metadata.Context
	baseDir := fmt.Sprintf("fluid-diagnose-%s", cfg.timestamp.Format("20060102-150405"))

	manifestBuilder := NewManifestBuilder("v1", cfg.timestamp)
	archiveWriter := NewArchiveWriter(baseDir, cfg.timestamp)

	// Redactor
	var redactor Redactor
	if cfg.redact {
		redactor = newRedactor()
	}

	// Helper to Process and Add a File
	addFile := func(path string, data interface{}, isJSON bool) error {
		var content []byte
		var err error

		// Redaction phase
		if cfg.redact {
			// If it's a map/struct, use structural redaction
			// If it's bytes, use regex
			switch v := data.(type) {
			case []byte:
				content = redactor.Redact(v)
			case string:
				content = []byte(redactor.RedactString(v))
			default:
				// Structural redaction for JSON/YAML objects
				cleanData, err := scrubJSON(data)
				if err != nil {
					return fmt.Errorf("redaction failed for %s: %w", path, err)
				}
				data = cleanData
			}
		}

		// Serialization phase (if not already bytes)
		if content == nil {
			if b, ok := data.([]byte); ok {
				content = b
			} else if s, ok := data.(string); ok {
				content = []byte(s)
			} else {
				if isJSON {
					content, err = json.MarshalIndent(data, "", "  ")
				} else {
					content, err = yaml.Marshal(data)
				}
				if err != nil {
					return fmt.Errorf("serialization failed for %s: %w", path, err)
				}
			}
		}

		// Add to manifest and writer
		manifestBuilder.AddFile(path, int64(len(content)), content)
		archiveWriter.AddFile(path, content)
		return nil
	}

	// 3. Add Core Files

	// Graph
	if err := addFile(GraphFile, input.Graph, true); err != nil {
		return nil, err
	}

	// Diagnosis
	if err := addFile(DiagnosisFile, input.Diagnosis, true); err != nil {
		return nil, err
	}

	// Metadata
	// Split metadata into smaller files if needed, or keep as one?
	// The design says metadata/environment.json, metadata/version.json
	// But input.Metadata is a single struct. We can marshal it to environment.json for now.
	if err := addFile(filepath.Join(MetadataDir, "environment.json"), input.Metadata, true); err != nil {
		return nil, err
	}

	// Summary (Text)
	summary := fmt.Sprintf("Fluid Diagnostic Bundle\nGenerated: %s\nIssues: %d\n",
		cfg.timestamp.Format(time.RFC3339), len(input.Diagnosis.Issues))
	if err := addFile(SummaryFile, summary, false); err != nil {
		return nil, err
	}

	// 4. Add Logs
	for filename, content := range input.Logs {
		if err := addFile(filepath.Join(LogsDir, filename), content, false); err != nil {
			return nil, err
		}
	}

	// 5. Add Raw Resources
	for path, content := range input.Resources {
		// Assuming content is YAML string or bytes
		if err := addFile(filepath.Join(ResourcesDir, path), content, false); err != nil {
			return nil, err
		}
	}

	// 6. Finalize Manifest
	manifest := manifestBuilder.Build()
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}
	// We add manifest to archive, but NOT to the manifest builder (recursion)
	archiveWriter.AddFile(ManifestFile, manifestBytes)

	// 7. Write Archive
	archivePath, size, err := archiveWriter.WriteToDisk(cfg.outputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to write archive: %w", err)
	}

	return &types.BundleResult{
		ArchivePath: archivePath,
		FileCount:   manifest.TotalFiles + 1, // +1 for manifest.json
		Manifest:    manifest,
		SizeBytes:   size,
	}, nil
}
