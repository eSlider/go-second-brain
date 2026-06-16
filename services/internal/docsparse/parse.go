package docsparse

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/eSlider/go-second-brain/services/internal/idkind"
)

var (
	reMDLink      = regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`)
	rePainLine    = regexp.MustCompile(`\*\*\[(PAIN-\d{2})]`)
	reAutoHeader  = regexp.MustCompile(`(?m)^##\s+(AUTO-\d{2})\b`)
	reAgentHeader = regexp.MustCompile(`(?m)^##\s+(AGENT-\d{2})\b`)
	reRoadHeader  = regexp.MustCompile(`(?m)^##\s+(ROAD-\d{2})\b`)
)

// ParseFile parses a single markdown file relative to docs/project root.
func ParseFile(rel string, data []byte) (*ParseResult, error) {
	rel = filepath.ToSlash(rel)
	text := string(data)
	kind, pathID := idkind.ClassifyPath(rel)
	switch {
	case kind == idkind.KindSubject && filepath.Base(rel) != "README.md":
		return parseSubject(rel, text)
	case kind == idkind.KindUseCase:
		return parseUseCase(rel, text)
	case kind == idkind.KindProcess:
		return parseProcess(rel, text, pathID)
	case rel == "optimization/pain-points.md":
		return parsePainPointsFile(rel, text)
	case rel == "optimization/automation-opportunities.md":
		return parseSectionedByHeader(rel, text, reAutoHeader, idkind.KindAutomation)
	case rel == "optimization/ai-agent-ideas.md":
		return parseSectionedByHeader(rel, text, reAgentHeader, idkind.KindAgentIdea)
	case rel == "optimization/roadmap.md":
		return parseSectionedByHeader(rel, text, reRoadHeader, idkind.KindRoadmap)
	case rel == "glossary.md":
		return parseGlossary(rel, text)
	default:
		return parseGenericDoc(rel, text, pathID)
	}
}

func parseSubject(rel, text string) (*ParseResult, error) {
	title := firstLineTitle(text)
	id := extractSubjectID(title)
	if id == "" {
		ids := idkind.ExtractInlineIDs(text)
		if len(ids) > 0 {
			id = ids[0]
		}
	}
	if id == "" {
		return nil, fmt.Errorf("docsparse: cannot determine SUBJ id in %s", rel)
	}
	n := Node{
		ID:      id,
		Kind:    idkind.KindSubject,
		Title:   stripTitleSuffix(title),
		Path:    rel,
		Summary: firstParagraph(text),
		Body:    text,
	}
	return buildResult(n, text, rel)
}

func parseUseCase(rel, text string) (*ParseResult, error) {
	base := strings.TrimSuffix(filepath.Base(rel), ".md")
	parts := strings.SplitN(base, "-", 3)
	if len(parts) < 2 {
		return nil, fmt.Errorf("docsparse: bad UC filename %s", rel)
	}
	ucID := parts[0] + "-" + parts[1]
	title := firstLineTitle(text)
	n := Node{
		ID:      ucID,
		Kind:    idkind.KindUseCase,
		Title:   stripTitleSuffix(title),
		Path:    rel,
		Summary: firstParagraph(text),
		Body:    text,
	}
	return buildResult(n, text, rel)
}

func parseProcess(rel, text, procID string) (*ParseResult, error) {
	title := firstLineTitle(text)
	n := Node{
		ID:      procID,
		Kind:    idkind.KindProcess,
		Title:   stripTitleSuffix(title),
		Path:    rel,
		Summary: firstParagraph(text),
		Body:    text,
	}
	return buildResult(n, text, rel)
}

func parsePainPointsFile(rel, text string) (*ParseResult, error) {
	var nodes []Node
	var edges []Edge
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		m := rePainLine.FindStringSubmatch(line)
		if len(m) < 2 {
			continue
		}
		pid := m[1]
		summary := strings.TrimSpace(strings.TrimPrefix(line, "- "))
		if idx := strings.Index(summary, "—"); idx >= 0 {
			summary = strings.TrimSpace(summary[idx+len("—"):])
		}
		nodes = append(nodes, Node{
			ID:      pid,
			Kind:    idkind.KindPain,
			Title:   pid,
			Path:    rel + "#" + strings.ToLower(pid),
			Summary: clip(summary, 240),
			Body:    line,
		})
		for _, ref := range idkind.ExtractInlineIDs(line) {
			if ref == pid {
				continue
			}
			edges = append(edges, Edge{FromID: pid, ToID: ref, Rel: "REFS"})
			applyHeuristicEdges(&edges, pid, ref)
		}
	}
	var chunks []TextChunk
	for _, n := range nodes {
		chunks = append(chunks, chunkNode(n, text, rel)...)
	}
	return &ParseResult{Nodes: nodes, Edges: dedupeEdges(edges), Chunks: chunks}, nil
}

func parseSectionedByHeader(rel, text string, header *regexp.Regexp, kind idkind.Kind) (*ParseResult, error) {
	indices := header.FindAllStringSubmatchIndex(text, -1)
	if len(indices) == 0 {
		return parseGenericDoc(rel, text, "DOC:"+rel)
	}
	var nodes []Node
	var edges []Edge
	for i, loc := range indices {
		id := text[loc[2]:loc[3]]
		start := loc[0]
		end := len(text)
		if i+1 < len(indices) {
			end = indices[i+1][0]
		}
		section := strings.TrimSpace(text[start:end])
		titleLine, rest, _ := strings.Cut(section, "\n")
		summary := ""
		if strings.TrimSpace(rest) != "" {
			summary = firstParagraph(rest)
		}
		n := Node{
			ID:      id,
			Kind:    kind,
			Title:   strings.TrimSpace(strings.TrimPrefix(titleLine, "##")),
			Path:    rel + "#" + strings.ToLower(id),
			Summary: summary,
			Body:    section,
		}
		nodes = append(nodes, n)
		for _, ref := range idkind.ExtractInlineIDs(section) {
			if ref == id {
				continue
			}
			edges = append(edges, Edge{FromID: id, ToID: ref, Rel: "REFS"})
			applyHeuristicEdges(&edges, id, ref)
		}
		for _, l := range extractLinkTargets(section) {
			tid := targetToID(l)
			if tid == "" || tid == id {
				continue
			}
			edges = append(edges, Edge{FromID: id, ToID: tid, Rel: "REFS"})
			applyHeuristicEdges(&edges, id, tid)
		}
	}
	var chunks []TextChunk
	for _, n := range nodes {
		chunks = append(chunks, chunkNode(n, n.Body, rel)...)
	}
	return &ParseResult{Nodes: nodes, Edges: dedupeEdges(edges), Chunks: chunks}, nil
}

func parseGlossary(rel, text string) (*ParseResult, error) {
	var nodes []Node
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- **") {
			continue
		}
		rest := strings.TrimPrefix(line, "- **")
		termDE, after, ok := strings.Cut(rest, "**")
		if !ok {
			continue
		}
		after = strings.TrimSpace(after)
		if !strings.HasPrefix(after, "—") {
			continue
		}
		desc := strings.TrimSpace(strings.TrimPrefix(after, "—"))
		anchor := ""
		if idx := strings.Index(desc, "{:"); idx >= 0 {
			meta := desc[idx:]
			desc = strings.TrimSpace(desc[:idx])
			if i := strings.Index(meta, "#"); i >= 0 {
				end := strings.IndexAny(meta[i+1:], " }")
				if end < 0 {
					anchor = strings.Trim(meta[i+1:], " }")
				} else {
					anchor = meta[i+1 : i+1+end]
				}
			}
		}
		if anchor == "" {
			continue
		}
		id := "TERM:" + anchor
		nodes = append(nodes, Node{
			ID:      id,
			Kind:    idkind.KindTerm,
			Title:   strings.TrimSpace(termDE),
			Path:    rel + "#" + anchor,
			Summary: clip(desc, 400),
			Body:    line,
		})
	}
	var chunks []TextChunk
	for _, n := range nodes {
		chunks = append(chunks, chunkNode(n, n.Body, rel)...)
	}
	return &ParseResult{Nodes: nodes, Edges: nil, Chunks: chunks}, nil
}

func parseGenericDoc(rel, text, docID string) (*ParseResult, error) {
	title := firstLineTitle(text)
	if docID == "" {
		docID = "DOC:" + rel
	}
	n := Node{
		ID:      docID,
		Kind:    idkind.KindDocument,
		Title:   stripTitleSuffix(title),
		Path:    rel,
		Summary: firstParagraph(text),
		Body:    text,
	}
	return buildResult(n, text, rel)
}

func buildResult(n Node, fullText, rel string) (*ParseResult, error) {
	var edges []Edge
	for _, ref := range idkind.ExtractInlineIDs(fullText) {
		if ref == n.ID {
			continue
		}
		edges = append(edges, Edge{FromID: n.ID, ToID: ref, Rel: "REFS"})
		applyHeuristicEdges(&edges, n.ID, ref)
	}
	for _, l := range extractLinkTargets(fullText) {
		tid := targetToID(l)
		if tid == "" || tid == n.ID {
			continue
		}
		edges = append(edges, Edge{FromID: n.ID, ToID: tid, Rel: "REFS"})
		applyHeuristicEdges(&edges, n.ID, tid)
	}
	chunks := chunkNode(n, fullText, rel)
	return &ParseResult{Nodes: []Node{n}, Edges: dedupeEdges(edges), Chunks: chunks}, nil
}

func applyHeuristicEdges(edges *[]Edge, from, to string) {
	fk := idkind.KindFromID(from)
	tk := idkind.KindFromID(to)
	switch {
	case fk == idkind.KindUseCase && tk == idkind.KindSubject:
		*edges = append(*edges, Edge{FromID: from, ToID: to, Rel: "INVOLVES"})
	case fk == idkind.KindProcess && tk == idkind.KindUseCase:
		*edges = append(*edges, Edge{FromID: from, ToID: to, Rel: "CONTAINS"})
	case fk == idkind.KindPain && (tk == idkind.KindUseCase || tk == idkind.KindSubject || tk == idkind.KindProcess):
		*edges = append(*edges, Edge{FromID: from, ToID: to, Rel: "AFFECTS"})
	case fk == idkind.KindAutomation && tk == idkind.KindPain:
		*edges = append(*edges, Edge{FromID: from, ToID: to, Rel: "ADDRESSES"})
	case fk == idkind.KindAgentIdea && (tk == idkind.KindPain || tk == idkind.KindUseCase):
		*edges = append(*edges, Edge{FromID: from, ToID: to, Rel: "AUTOMATES"})
	case fk == idkind.KindRoadmap && (tk == idkind.KindAutomation || tk == idkind.KindAgentIdea):
		*edges = append(*edges, Edge{FromID: from, ToID: to, Rel: "DELIVERS"})
	}
}

func extractLinkTargets(text string) []string {
	var out []string
	for _, m := range reMDLink.FindAllStringSubmatch(text, -1) {
		if len(m) < 3 {
			continue
		}
		out = append(out, m[2])
	}
	return out
}

func targetToID(target string) string {
	target = strings.TrimSpace(target)
	target = strings.Split(target, "#")[0]
	target = strings.TrimPrefix(target, "./")
	base := filepath.Base(target)
	base = strings.TrimSuffix(base, ".md")
	if strings.HasPrefix(base, "UC-") {
		parts := strings.SplitN(base, "-", 3)
		if len(parts) >= 2 {
			return parts[0] + "-" + parts[1]
		}
	}
	return ""
}

var reSubjectID = regexp.MustCompile(`SUBJ-[A-Z0-9-]+`)

func extractSubjectID(title string) string {
	title = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(title), "#"))
	if m := reSubjectID.FindString(title); m != "" {
		return m
	}
	return ""
}

func firstLineTitle(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "---") {
			continue
		}
		if strings.HasPrefix(line, "#") {
			return strings.TrimSpace(strings.TrimLeft(line, "#"))
		}
	}
	return ""
}

func stripTitleSuffix(title string) string {
	title = strings.TrimSpace(title)
	if i := strings.Index(title, "—"); i > 0 {
		return strings.TrimSpace(title[:i])
	}
	return title
}

func firstParagraph(text string) string {
	var buf strings.Builder
	in := false
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			if in {
				break
			}
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		in = true
		if buf.Len() > 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(line)
		if buf.Len() > 380 {
			break
		}
	}
	return clip(buf.String(), 500)
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func dedupeEdges(edges []Edge) []Edge {
	seen := make(map[string]struct{})
	var out []Edge
	for _, e := range edges {
		k := e.FromID + "|" + e.Rel + "|" + e.ToID
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, e)
	}
	return out
}
