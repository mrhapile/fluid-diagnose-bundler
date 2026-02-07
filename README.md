# Fluid Diagnose Bundler

A reusable, zero-dependency Go library for creating deterministic, redacted, and structured diagnostic archives for Fluid.

## Overview

The `fluid-diagnose-bundler` is a specialized library designed to take in raw diagnostic data (Kubernetes resources, logs, analysis results) and package them into a portable `.tar.gz` archive.

This library is the canonical implementation of the Fluid Diagnostic Format (Phase 0 Design). It is used by `fluidctl` and can be integrated into CI systems, operators, and future AI-driven analysis tools.

## Features

*   **Deterministic**: Produces bitwise identical archives for the same input (sorted files, stable JSON, controlled timestamps).
*   **Secure by Default**: Built-in redaction pipeline for secrets, tokens, and sensitive keys.
*   **Structured**: Generates a `manifest.json` index and organizes files in a strict hierarchy for machine consumption.
*   **Offline-First**: Does NOT connect to Kubernetes. It expects data to be passed in, making it testable and safe to run anywhere.


## Integrations

### fluidctl

This library powers the `fluidctl diagnose dataset --archive` command. 

```bash
fluidctl diagnose dataset demo-data --archive
# Output: fluid-diagnose-demo-data-2026xxxx.tar.gz
```

By decoupling the bundler from the CLI:
1.  **Determinism**: The CLI guarantees consistent output across different user machines (Mac/Linux/Windows).
2.  **Safety**: The library handles redaction centrally, preventing accidental leak of secrets via ad-hoc tar commands.
3.  **Reuse**: The same bundler logic can be imported by the Fluid Operator or CI test runners.

## Usage

```go
package main

import (
	"time"
	"github.com/mrhapile/fluid-diagnose-bundler/pkg/bundler"
	"github.com/mrhapile/fluid-diagnose-bundler/pkg/types"
)

func main() {
	// 1. Gather Data (upstream logic)
	graph := map[string]interface{}{"kind": "Dataset", "name": "demo"}
	diagnosis := types.DiagnosticResult{Issues: []types.Issue{{Level: "Warn", Message: "Check logs"}}}
	
	input := types.BundleInput{
		Graph:     graph,
		Diagnosis: diagnosis,
		Metadata:  types.BundleMetadata{Environment: "production"},
		Logs:      map[string][]byte{"operator.log": []byte("error: connection refused")},
	}

	// 2. Build Bundle
	result, err := bundler.Build(input,
		bundler.WithRedaction(),
		bundler.WithTimestamp(time.Now()), // OR use fixed time for determinism
		bundler.WithOutputDir("/tmp"),
	)
	if err != nil {
		panic(err)
	}

	println("Bundle created at:", result.ArchivePath)
}
```

## Non-Goals

This library explicitly does **NOT**:
*   Execute `kubectl` or interact with the Kubernetes API.
*   Make network requests.
*   contain CLI flags or `cobra` commands.
*   Perform AI inference (it prepares data *for* AI).

## Why not just use `tar`?

Ad-hoc shell scripts using `tar` are fragile:
1.  **Non-deterministic**: File order and headers vary by OS and `tar` version, breaking checksum verification.
2.  **Unstructured**: "Dump and zip" makes it hard for automated tools to parse the content reliability.
3.  **Insecure**: It's easy to accidentally include secrets without a dedicated redaction pass.

This library solves these problems by enforcing a schema and a rigorous build pipeline.
