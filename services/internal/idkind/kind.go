// Package idkind classifies stable document and entity IDs from paths and text.
package idkind

import (
	"path/filepath"
	"regexp"
	"strings"
)

// Kind is a Neo4j label category for ingested nodes.
type Kind string

const (
	KindSubject    Kind = "Subject"
	KindUseCase    Kind = "UseCase"
	KindPain       Kind = "Pain"
	KindAutomation Kind = "Automation"
	KindAgentIdea  Kind = "AgentIdea"
	KindRoadmap    Kind = "RoadmapItem"
	KindProcess    Kind = "Process"
	KindTerm       Kind = "Term"
	KindDocument   Kind = "Document"
)

var (
	reUC        = regexp.MustCompile(`^UC-\d{2}$`)
	rePain      = regexp.MustCompile(`^PAIN-\d{2}$`)
	reAuto      = regexp.MustCompile(`^AUTO-\d{2}$`)
	reAgent     = regexp.MustCompile(`^AGENT-\d{2}$`)
	reRoad      = regexp.MustCompile(`^ROAD-\d{2}$`)
	reSubject   = regexp.MustCompile(`^SUBJ-[A-Z0-9-]+$`)
	reInlineIDs = regexp.MustCompile(`\b(SUBJ-[A-Z0-9-]+|UC-\d{2}|PAIN-\d{2}|AUTO-\d{2}|AGENT-\d{2}|ROAD-\d{2})\b`)
)

// KindFromID returns the Kind for a stable ID string, or empty if unknown.
func KindFromID(id string) Kind {
	id = strings.TrimSpace(id)
	switch {
	case reSubject.MatchString(id):
		return KindSubject
	case reUC.MatchString(id):
		return KindUseCase
	case rePain.MatchString(id):
		return KindPain
	case reAuto.MatchString(id):
		return KindAutomation
	case reAgent.MatchString(id):
		return KindAgentIdea
	case reRoad.MatchString(id):
		return KindRoadmap
	default:
		return ""
	}
}

// ExtractInlineIDs finds SUBJ/UC/PAIN/AUTO/AGENT/ROAD tokens in text.
func ExtractInlineIDs(text string) []string {
	found := reInlineIDs.FindAllString(text, -1)
	uniq := make(map[string]struct{})
	var out []string
	for _, s := range found {
		if _, ok := uniq[s]; ok {
			continue
		}
		uniq[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// ClassifyPath returns Kind and primary ID hints from a path relative to the corpus root.
func ClassifyPath(rel string) (kind Kind, primaryID string) {
	rel = filepath.ToSlash(rel)
	base := filepath.Base(rel)
	if base == "README.md" {
		return KindDocument, ""
	}
	switch {
	case strings.HasPrefix(rel, "subjects/") && strings.HasSuffix(rel, ".md"):
		return KindSubject, ""
	case strings.HasPrefix(rel, "cases/") && strings.HasPrefix(base, "UC-"):
		if i := strings.Index(base, "-"); i > 0 {
			// UC-07-monthly-billing.md -> UC-07
			parts := strings.Split(base[:len(base)-3], "-")
			if len(parts) >= 2 {
				return KindUseCase, parts[0] + "-" + parts[1]
			}
		}
		return KindUseCase, ""
	case strings.HasPrefix(rel, "processes/") && strings.HasSuffix(rel, ".md"):
		stem := strings.TrimSuffix(base, ".md")
		return KindProcess, "PROC-" + stem
	case rel == "optimization/pain-points.md":
		return KindPain, ""
	case rel == "optimization/automation-opportunities.md":
		return KindAutomation, ""
	case rel == "optimization/ai-agent-ideas.md":
		return KindAgentIdea, ""
	case rel == "optimization/roadmap.md":
		return KindRoadmap, ""
	case rel == "glossary.md":
		return KindTerm, ""
	default:
		return KindDocument, "DOC:" + rel
	}
}
