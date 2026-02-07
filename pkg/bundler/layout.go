package bundler

import (
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/mrhapile/fluid-diagnose-bundler/pkg/types"
)

const (
	ManifestFile  = "manifest.json"
	SummaryFile   = "summary.txt"
	GraphFile     = "graph.json"
	DiagnosisFile = "diagnosis.json"
	ResourcesDir  = "resources"
	LogsDir       = "logs"
	MetadataDir   = "metadata"
)

// Layout holds the mapping of logical content to archive paths.
type Layout struct {
	BaseDir string
	Files   map[string]interface{} // path -> content
}

func newLayout(datasetName string, ts time.Time) *Layout {
	// Base directory for the tarball content
	baseDir := fmt.Sprintf("fluid-diagnose-%s-%s", datasetName, ts.Format("20060102-150405"))
	return &Layout{
		BaseDir: baseDir,
		Files:   make(map[string]interface{}),
	}
}

func (l *Layout) addFile(path string, content interface{}) {
	l.Files[filepath.Join(l.BaseDir, path)] = content
}

func (l *Layout) getSortedPaths() []string {
	paths := make([]string, 0, len(l.Files))
	for k := range l.Files {
		paths = append(paths, k)
	}
	sort.Strings(paths)
	return paths
}

// Helper to determine dataset name from graph or metadata if not provided
func extractDatasetName(graph types.ResourceGraph) string {
	// Heuristic: check if graph has a root dataset object
	// For now, default to "unknown" if not found
	// In a real implementation this would parse the graph
	return "unknown"
}
