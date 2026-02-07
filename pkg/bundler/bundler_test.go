package bundler_test

import (
	"os"
	"testing"
	"time"

	"github.com/mrhapile/fluid-diagnose-bundler/pkg/bundler"
	"github.com/mrhapile/fluid-diagnose-bundler/pkg/types"
)

func TestBuild(t *testing.T) {
	// Create temp output dir
	outDir, err := os.MkdirTemp("", "fluid-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outDir)

	// Mock Input
	input := types.BundleInput{
		Graph: types.ResourceGraph{
			"kind": "Dataset",
			"metadata": map[string]string{
				"name": "demo",
			},
		},
		Diagnosis: types.DiagnosticResult{
			Issues: []types.Issue{},
		},
		Metadata: types.BundleMetadata{
			Environment:       "test",
			CreationTimestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		Logs: map[string][]byte{
			"test.log": []byte("test content"),
		},
		Resources: map[string]string{
			"dataset.yaml": "kind: Dataset\n",
		},
	}

	// Use fixed timestamp for determinism
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	result, err := bundler.Build(input,
		bundler.WithRedaction(),
		bundler.WithTimestamp(fixedTime),
		bundler.WithOutputDir(outDir),
	)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Basic assertions
	if result.ArchivePath == "" {
		t.Error("ArchivePath is empty")
	}
	if result.FileCount <= 0 {
		t.Errorf("FileCount %d <= 0", result.FileCount)
	}
	if result.Manifest.GeneratedAt != fixedTime {
		t.Error("Manifest timestamp mismatch")
	}

	// Verify file existence
	if _, err := os.Stat(result.ArchivePath); os.IsNotExist(err) {
		t.Errorf("Archive not found: %s", result.ArchivePath)
	}
}
