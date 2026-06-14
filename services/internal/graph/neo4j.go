// Package graph writes and reads the knowledge graph in Neo4j.
package graph

import (
	"context"
	"fmt"
	"strings"

	"git.produktor.io/edelweiss/docs/services/internal/docsparse"
	"git.produktor.io/edelweiss/docs/services/internal/idkind"
	neon "git.produktor.io/edelweiss/docs/services/pkg/neo4j"

	driver "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Store wraps a Neo4j driver sourced from pkg/neo4j.
type Store struct {
	neo *neon.Client
}

// NewStore opens a Neo4j driver via pkg/neo4j (bolt URI).
func NewStore(ctx context.Context, uri, user, password string) (*Store, error) {
	c, err := neon.New(ctx, &neon.Config{URI: uri, User: user, Password: password})
	if err != nil {
		return nil, fmt.Errorf("graph: %w", err)
	}
	return &Store{neo: c}, nil
}

// Close closes the Neo4j driver.
func (s *Store) Close(ctx context.Context) error {
	_ = ctx
	if s.neo == nil {
		return nil
	}
	return s.neo.Close()
}

// Driver exposes the neo4j-go-driver handle for advanced calls.
func (s *Store) Driver() driver.DriverWithContext {
	if s.neo == nil {
		return nil
	}
	return s.neo.Driver
}

// WriteCorpus merges nodes and edges.
func (s *Store) WriteCorpus(ctx context.Context, pr *docsparse.ParseResult) error {
	sess := s.Driver().NewSession(ctx, driver.SessionConfig{AccessMode: driver.AccessModeWrite})
	defer func() {
		_ = sess.Close(ctx)
	}()
	_, err := sess.ExecuteWrite(ctx, func(tx driver.ManagedTransaction) (any, error) {
		for _, n := range pr.Nodes {
			lbl, ok := labelFor(n.Kind)
			if !ok {
				continue
			}
			cypher := fmt.Sprintf(
				`MERGE (x:%s {id: $id}) SET x.title = $title, x.path = $path, x.summary = $summary`,
				lbl,
			)
			params := map[string]any{
				"id":      n.ID,
				"title":   n.Title,
				"path":    n.Path,
				"summary": n.Summary,
			}
			if _, err := tx.Run(ctx, cypher, params); err != nil {
				return nil, fmt.Errorf("graph: merge node %s: %w", n.ID, err)
			}
		}
		for _, e := range pr.Edges {
			rel := sanitizeRel(e.Rel)
			cypher := fmt.Sprintf(`
MATCH (a {id: $from}), (b {id: $to})
MERGE (a)-[r:%s]->(b)
`, rel)
			if _, err := tx.Run(ctx, cypher, map[string]any{"from": e.FromID, "to": e.ToID}); err != nil {
				return nil, fmt.Errorf("graph: merge edge %s-%s: %w", e.FromID, e.ToID, err)
			}
		}
		return nil, nil
	})
	return err
}

func labelFor(k idkind.Kind) (string, bool) {
	switch k {
	case idkind.KindSubject:
		return "Subject", true
	case idkind.KindUseCase:
		return "UseCase", true
	case idkind.KindPain:
		return "Pain", true
	case idkind.KindAutomation:
		return "Automation", true
	case idkind.KindAgentIdea:
		return "AgentIdea", true
	case idkind.KindRoadmap:
		return "RoadmapItem", true
	case idkind.KindProcess:
		return "Process", true
	case idkind.KindTerm:
		return "Term", true
	case idkind.KindDocument:
		return "Document", true
	default:
		return "", false
	}
}

var allowedRels = map[string]struct{}{
	"REFS": {}, "INVOLVES": {}, "CONTAINS": {}, "AFFECTS": {}, "ADDRESSES": {}, "AUTOMATES": {}, "DELIVERS": {},
}

func sanitizeRel(r string) string {
	r = strings.TrimSpace(r)
	if _, ok := allowedRels[r]; ok {
		return r
	}
	return "REFS"
}
