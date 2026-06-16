package docsparse

import "github.com/eSlider/go-second-brain/services/internal/idkind"

// Node is a graph entity extracted from documentation.
type Node struct {
	ID      string
	Kind    idkind.Kind
	Title   string
	Path    string
	Summary string
	Body    string
}

// Edge is a directed relationship between stable IDs.
type Edge struct {
	FromID string
	ToID   string
	Rel    string
}

// TextChunk is a heading-scoped slice of markdown for embedding.
type TextChunk struct {
	NodeID  string
	Heading string
	Path    string
	Text    string
}

// ParseResult aggregates nodes, edges, and chunks from the corpus.
type ParseResult struct {
	Nodes  []Node
	Edges  []Edge
	Chunks []TextChunk
}
