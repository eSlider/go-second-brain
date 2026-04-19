package rag

import (
	"regexp"
	"strings"
)

var (
	reGarbageIntro = regexp.MustCompile(`(?i)^(вот\s+)?(анализ|результаты)(\s+предоставленного)?`)
	reIDPrefixLine = regexp.MustCompile(`(?i)^\s*(SUBJ-|PAIN-|AUTO-|AGENT-|ROAD-|PROC-|PB-|CD-|DOC:)`)
	reUCDumpLine   = regexp.MustCompile(`^UC-\d+:`)
)

// sanitizeBotAnswer strips model slop: identifier dumps, graph edge lines, "used IDs" sections.
func sanitizeBotAnswer(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	// Cut everything from "Использованные идентификаторы" onward (case-insensitive).
	if i := strings.Index(strings.ToLower(s), "использованные идентификаторы"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	lines := strings.Split(s, "\n")
	var out []string
	skipIntro := true
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if skipIntro {
			if t == "" {
				continue
			}
			if reGarbageIntro.MatchString(t) {
				continue
			}
			skipIntro = false
		}
		if isIdentifierDumpLine(t) {
			continue
		}
		out = append(out, line)
	}
	s = strings.TrimSpace(strings.Join(out, "\n"))
	s = strings.TrimRight(s, "\n")
	return strings.TrimSpace(s)
}

func isIdentifierDumpLine(t string) bool {
	if t == "" {
		return false
	}
	if reIDPrefixLine.MatchString(t) {
		return true
	}
	if strings.Contains(t, "INVOLVES") || strings.Contains(t, "Involves") {
		return true
	}
	// Long UC-* lines are almost always retrieval dumps, not a human answer.
	if reUCDumpLine.MatchString(t) && (len(t) > 90 || strings.Contains(t, "Отражает") ||
		strings.Contains(t, "Указывает на процесс") || strings.Contains(t, "Аналогично")) {
		return true
	}
	return false
}
