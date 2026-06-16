// Package rag orchestrates retrieval and generation.
package rag

import (
	"context"
	"fmt"
	"strings"

	"github.com/eSlider/go-second-brain/services/pkg/ollama"
	"github.com/eSlider/go-second-brain/services/pkg/qdrant"
)

// Engine ties stores and models.
type Engine struct {
	Ollama       *ollama.Client
	Qdrant       *qdrant.Client
	EmbedModel   string
	GenModel     string
	Collection   string
	TopK         int
	SystemPrompt string
}

// Answer returns an answer grounded in retrieved chunks.
func (e *Engine) Answer(ctx context.Context, userQuery string) (string, error) {
	if e.TopK <= 0 {
		e.TopK = 8
	}
	qv, err := e.Ollama.Embed(ctx, e.EmbedModel, userQuery)
	if err != nil {
		return "", fmt.Errorf("rag: embed query: %w", err)
	}
	hits, err := e.Qdrant.Search(ctx, e.Collection, qv, e.TopK)
	if err != nil {
		return "", fmt.Errorf("rag: search: %w", err)
	}
	var ctxParts []string
	seen := map[string]struct{}{}
	for _, h := range hits {
		nodeID, _ := h.Payload["node_id"].(string)
		text, _ := h.Payload["text"].(string)
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		if nodeID != "" {
			if _, ok := seen[nodeID]; ok {
				continue
			}
			seen[nodeID] = struct{}{}
		}
		ctxParts = append(ctxParts, text)
	}
	contextBlock := strings.Join(ctxParts, "\n\n---\n\n")
	system := strings.TrimSpace(e.SystemPrompt)
	prompt := strings.Join([]string{
		"Answer the question using only the excerpts below.",
		"",
		"Question:",
		userQuery,
		"",
		"Excerpts:",
		contextBlock,
	}, "\n")
	raw, err := e.Ollama.GenerateWithSystem(ctx, e.GenModel, system, prompt)
	if err != nil {
		return "", err
	}
	return sanitizeBotAnswer(raw), nil
}

// BuildEngineFromConfig wires shared Ollama + Qdrant accessors for cmds.
func BuildEngineFromConfig(
	ollamaClient *ollama.Client,
	qdr *qdrant.Client,
	embedModel, genModel, collection string,
	topK int,
	systemPrompt string,
) *Engine {
	if topK <= 0 {
		topK = 8
	}
	return &Engine{
		Ollama:       ollamaClient,
		Qdrant:       qdr,
		EmbedModel:   embedModel,
		GenModel:     genModel,
		Collection:   collection,
		TopK:         topK,
		SystemPrompt: systemPrompt,
	}
}
