package types

import "time"

// ResourceGraph represents the connected state of Kubernetes and Fluid resources.
// It is expected to be a map of resource kinds to lists of resources, or a more complex graph object.
// For the bundler's purpose, it's treated as a generic JSON-serializable object.
type ResourceGraph map[string]interface{}

// DiagnosticResult represents the outcome of a diagnostic run.
type DiagnosticResult struct {
	Issues []Issue `json:"issues"`
	Score  int     `json:"score"`
}

// Issue represents a single finding in the diagnosis.
type Issue struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

// BundleMetadata contains contextual information about the diagnosis environment.
type BundleMetadata struct {
	CreationTimestamp time.Time `json:"creationTimestamp"`
	FluidVersion      string    `json:"fluidVersion"`
	K8sVersion        string    `json:"k8sVersion"`
	Environment       string    `json:"environment,omitempty"` // e.g., "production", "ci"
}

// BundleInput is the input payload for creating a diagnostic bundle.
type BundleInput struct {
	Graph     ResourceGraph     // The raw resource state
	Diagnosis DiagnosticResult  // The computed diagnosis
	Metadata  BundleMetadata    // Contextual metadata
	Logs      map[string][]byte // Log file contents (filename -> content)
	Resources map[string]string // Additional raw resource YAMLs (path -> content) if needed
}

// BundleResult represents the output of a successful bundling operation.
type BundleResult struct {
	ArchivePath string         // The absolute path to the generated archive file
	FileCount   int            // Total number of files archived
	Manifest    BundleManifest // The manifest generated for the archive
	SizeBytes   int64          // Size of the archive in bytes
}
