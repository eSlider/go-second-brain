package docsparse

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// WalkDocs walks root (typically examples/corpus), parses every .md file, and merges results.
func WalkDocs(root string) (*ParseResult, error) {
	root = filepath.Clean(root)
	var merged ParseResult
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "_scripts" || strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		// Path comes only from filepath.WalkDir under docs root (trusted tree).
		data, err := os.ReadFile(path) //nolint:gosec // G304 path is controlled by WalkDir
		if err != nil {
			return fmt.Errorf("docsparse: read %s: %w", path, err)
		}
		pr, err := ParseFile(rel, data)
		if err != nil {
			return fmt.Errorf("docsparse: parse %s: %w", rel, err)
		}
		merged.Nodes = append(merged.Nodes, pr.Nodes...)
		merged.Edges = append(merged.Edges, pr.Edges...)
		merged.Chunks = append(merged.Chunks, pr.Chunks...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	merged.Edges = dedupeEdges(merged.Edges)
	return &merged, nil
}
