# PHASE 0: Design Document - Fluid Diagnostic Bundler

## 1. Problem Statement

Current diagnostic collection in cloud-native ecosystems often relies on ad-hoc shell scripts or CLI-embedded logic (e.g., `kubectl cp`, `tar` pipes). These approaches significantly hurt reliability and maintainability:

*   **Fragility**: Shell scripts are platform-dependent and prone to breaking with tool version changes (e.g., BSD vs. GNU `tar`).
*   **Non-Determinism**: Archives created at different times or on different machines often produce different hashes even if the content is identical, breaking caching and verification.
*   **Lack of Structure**: "Dump everything" approaches make it impossible for automated tools (AI agents, CI parsers) to reliability consume the data without complex heuristics.
*   **Security Risks**: Ad-hoc scripts often miss redaction of sensitive data (Secrets, tokens, IP addresses) before creating the archive.

**Why a Library?**
Embedding this logic directly into a CLI command makes it hard to reuse in other contexts, such as:
*   Automated CI failure analysis.
*   In-cluster operator-triggered diagnostics.
*   AI-driven analysis pipelines which need structured input.

The `fluid-diagnose-bundler` aims to solve this by providing a **pure Go library** that guarantees a deterministic, structured, and secure diagnostic archive, serving as the canonical "export" format for Fluid diagnostics.

## 2. Design Principles

*   **Determinism**: The output archive MUST be bitwise identical for the same input, regardless of the time of execution (unless a timestamp is explicitly provided) or the operating system. This implies sorted file, specific tar header handling, and stable JSON marshaling.
*   **Read-Only**: The bundler purely consumes input data and produces an artifact. It never modifies the state of the cluster or the input objects.
*   **Redaction-First**: Security is not an afterthought. The design enforces a redacted-by-default approach where possible, or provides explicit hooks for redaction that run *before* data is written to the archive.
*   **Offline-First**: The library operates on in-memory Go structs (`ResourceGraph`, `DiagnosticResult`). It does NOT connect to Kubernetes, execute `kubectl`, or make network calls. It assumes data gathering is essentially "upstream" of this library.
*   **CLI-Agnostic**: The library creates a standard `.tar.gz` and returns metadata. It does not know or care about flags, `stdout` printing, or user interaction.

## 3. Input / Output Contract

The library defines a strict contract for inputs and outputs to ensure type safety and clear ownership.

### Type Definitions

```go
package types

import "time"

// ResourceGraph represents the collected state of Fluid resources.
// This is a placeholder for the actual heavy structs, assumed to be
// passed in fully populated.
type ResourceGraph map[string]interface{} // Simplified for design doc

// DiagnosticResult represents the analysis performed on the resources.
type DiagnosticResult struct {
	Issues []Issue `json:"issues"`
	Score  int     `json:"score"`
}

type Issue struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

// BundleMetadata contains environmental context for the diagnosis.
type BundleMetadata struct {
	CreationTime time.Time `json:"creationTimestamp"`
	FluidVersion string    `json:"fluidVersion"`
	K8sVersion   string    `json:"k8sVersion"`
	Environment  string    `json:"environment"` // e.g., "production", "ci"
}

// BundleInput is the primary entry point for the library.
type BundleInput struct {
	Graph     ResourceGraph    // The raw K8s resource state
	Diagnosis DiagnosticResult // The computed diagnosis
	Metadata  BundleMetadata   // Contextual metadata
	Logs      map[string][]byte // Log files (filename -> content)
}

// BundleManifest describes the contents of the generated archive.
type BundleManifest struct {
	Version      string            `json:"version"` // Schema version of the bundle
	GeneratedAt  time.Time         `json:"generatedAt"`
	TotalFiles   int               `json:"totalFiles"`
	Files        []FileEntry       `json:"files"`
	ContentHash  string            `json:"contentHash"` // SHA256 of the entire checkout
}

type FileEntry struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// BundleResult is returned to the caller after successful bundling.
type BundleResult struct {
	ArchivePath string          // Absolute path to the generated .tar.gz
	Manifest    BundleManifest  // The manifest that was written inside
	FileCount   int             // Validation helper
	sizeBytes   int64           // Validation helper
}
```

**Ownership**:
*   The **Caller** owns the `BundleInput` and is responsible for populating it (likely via `kubectl` calls or a separate collector library).
*   The **Library** owns the construction of the archive file and the generation of the `BundleManifest`.

## 4. Archive Layout Specification

The archive layout is fixed to support both human browsing and machine parsing.

### Structure

```text
fluid-diagnose-<dataset>-<timestamp>.tar.gz/
├── manifest.json              # The BundleManifest (Machine Readable Index)
├── summary.txt                # Human readable summary of the diagnosis
├── graph.json                 # serialized ResourceGraph (Machine Readable State)
├── diagnosis.json             # serialized DiagnosticResult (Machine Readable Analysis)
├── resources/                 # Raw YAML exports of involved resources
│   ├── dataset.yaml
│   ├── runtime.yaml
│   ├── configmaps/
│   │   └── ufs-conf.yaml
│   └── pods/
│       ├── fuse-0.yaml
│       └── worker-0.yaml
├── logs/                      # Collected container/pod logs
│   ├── master.log
│   ├── worker-0.log
│   └── fuse-0.log
└── metadata/                  # Contextual info
    ├── environment.json       # K8s version, platform info
    └── version.json           # Bundle schema version
```

### Rationale

*   **`manifest.json`**: Critical for AI agents and CI tools to "index" the content without reading every file. It provides a table of contents with checksums.
*   **`summary.txt`**: The "README" for a human debugger. Opens quickly, explains the high-level health.
*   **`resources/`**: Organized hierarchically to mimic a filesystem or K8s structure, making it intuitive for operators used to `kubectl get -o yaml`.
*   **`logs/`**: Separated to keep the root clean.
*   **`graph.json` & `diagnosis.json`**: These are the "API responses" frozen in time. Tools like a dashboard or a future `fluidctl analyze --from-bundle` would load these directly.

## 5. Security & Redaction Model

### Sensitive Data
*   **Secrets**: `Secret` resources should generally *not* be in `ResourceGraph`, but if they are, their `.data` fields must be redacted.
*   **Tokens**: ServiceAccount tokens in Pod specs.
*   **Environment Variables**: `env` sections in Pod specs which might contain keys.
*   **IP Addresses/Hostnames**: Internal network details (optional, but good practice).

### Strategy
1.  **Redaction Pipeline**: The bundler includes a `redact.go` component that acts as a middleware writer.
2.  **Pre-Write Processing**: Before any JSON/YAML is marshaled and added to the tarball, it passes through a sanitizer.
    *   Keywords like `password`, `token`, `key` in maps prompt value replacement with `[REDACTED]`.
    *   Specific struct fields (like `Secret.Data`) are wiped by type definition logic if robust typing is used, or key-traversal if using `map[string]interface{}`.
3.  **Extensibility**: The `Build()` function accepts options to add custom redaction rules (e.g., specific regex patterns).

## 6. Integration Points

### `fluidctl diagnose --archive`
The CLI command `fluidctl diag` will:
1.  Execute its existing logic to gather data (discovery, analysis).
2.  Populate `types.BundleInput`.
3.  Call `bundler.Build(input, bundler.WithOutputDir("."))`.
4.  Print "Diagnostic bundle saved to: ..."

### Benefits
*   **Maintenance**: If the archive format needs to change (e.g., adding a `metadata/` folder), it changes in *one* place (this library), not in every CLI command.
*   **AI Future-Proofing**: AI agents work best with structured, predictable inputs. By strictly defining the `manifest.json` and file hierarchy, we create a specialized "context window" payload for LLMs to analyze system health without needing access to the live cluster.
