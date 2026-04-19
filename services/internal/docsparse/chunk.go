package docsparse

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// chunkNode splits document text into heading-based chunks for embedding.
func chunkNode(n Node, fullText, rel string) []TextChunk {
	body := n.Body
	if strings.TrimSpace(body) == "" {
		body = fullText
	}
	sections := splitByHeadings(body)
	if len(sections) == 0 {
		return []TextChunk{{
			NodeID:  n.ID,
			Heading: "",
			Path:    rel,
			Text:    trimChunk(n.ID + "\n\n" + body),
		}}
	}
	var out []TextChunk
	for _, sec := range sections {
		out = append(out, TextChunk{
			NodeID:  n.ID,
			Heading: sec.heading,
			Path:    rel,
			Text:    trimChunk(n.ID + "\n" + sec.heading + "\n" + sec.body),
		})
	}
	return out
}

type section struct {
	heading string
	body    string
}

func splitByHeadings(text string) []section {
	lines := strings.Split(text, "\n")
	var sections []section
	var cur *section
	flush := func() {
		if cur == nil {
			return
		}
		if strings.TrimSpace(cur.body) != "" || cur.heading != "" {
			sections = append(sections, *cur)
		}
		cur = nil
	}
	var preamble strings.Builder
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "## ") && !strings.HasPrefix(t, "###") {
			flush()
			h := strings.TrimSpace(strings.TrimPrefix(t, "##"))
			cur = &section{heading: h}
			continue
		}
		if cur == nil {
			if preamble.Len() > 0 {
				preamble.WriteByte('\n')
			}
			preamble.WriteString(line)
			continue
		}
		if cur.body == "" && strings.TrimSpace(line) == "" {
			continue
		}
		cur.body += line + "\n"
	}
	flush()
	if preamble.Len() > 0 && len(sections) > 0 {
		sections = append([]section{{heading: "", body: preamble.String()}}, sections...)
	} else if preamble.Len() > 0 && len(sections) == 0 {
		sections = append(sections, section{heading: "", body: preamble.String()})
	}
	for i := range sections {
		sections[i].body = strings.TrimSpace(sections[i].body)
	}
	return sections
}

func trimChunk(s string) string {
	s = strings.TrimSpace(s)
	const maxRunes = 6000
	if len(s) <= maxRunes {
		return s
	}
	return s[:maxRunes] + "\n…"
}

// ChunkContentHash returns a stable hex hash for idempotent upserts.
func ChunkContentHash(c TextChunk) string {
	h := sha256.Sum256([]byte(c.Path + "\x00" + c.Heading + "\x00" + c.Text))
	return hex.EncodeToString(h[:])
}
