// Package rag orchestrates retrieval and generation.
package rag

import (
	"context"
	"fmt"
	"strings"

	"git.produktor.io/edelweiss/docs/services/internal/embed"
	"git.produktor.io/edelweiss/docs/services/internal/graph"
	"git.produktor.io/edelweiss/docs/services/internal/llm"
	"git.produktor.io/edelweiss/docs/services/internal/vectorstore"
)

// Engine ties stores and models.
type Engine struct {
	Embed      *embed.Client
	LLM        *llm.Client
	Qdrant     *vectorstore.Qdrant
	Graph      *graph.Store
	EmbedModel string
	GenModel   string
	Collection string
	TopK       int
}

// Answer returns a Russian answer grounded in retrieved chunks.
func (e *Engine) Answer(ctx context.Context, userQuery string) (string, error) {
	if e.TopK <= 0 {
		e.TopK = 8
	}
	qv, err := e.Embed.Embed(ctx, e.EmbedModel, userQuery)
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
		path, _ := h.Payload["path"].(string)
		text, _ := h.Payload["text"].(string)
		if nodeID != "" {
			if _, ok := seen[nodeID]; !ok {
				seen[nodeID] = struct{}{}
				nb, err := e.Graph.NeighborSummary(ctx, nodeID, 12)
				if err == nil && strings.TrimSpace(nb) != "" {
					ctxParts = append(ctxParts, fmt.Sprintf("Связи для %s: %s", nodeID, nb))
				}
			}
		}
		ctxParts = append(ctxParts, fmt.Sprintf("Фрагмент (%s, %s):\n%s", nodeID, path, text))
	}
	contextBlock := strings.Join(ctxParts, "\n\n")
	prompt := strings.Join([]string{
		"Ты ассистент по базе знаний Edelweiss Pflegedienst.",
		"Отвечай по-русски. Немецкие термины (Verordnung, Pflegegrad, Krankenkasse и т.д.) оставляй в оригинале.",
		"Опирайся только на предоставленный контекст. Если данных недостаточно — скажи об этом.",
		"В конце ответа перечисли использованные идентификаторы (SUBJ-*, UC-*, PAIN-*, AUTO-*, AGENT-*, ROAD-*) из контекста.",
		"",
		"Вопрос пользователя:",
		userQuery,
		"",
		"Контекст:",
		contextBlock,
	}, "\n")
	return e.LLM.Generate(ctx, e.GenModel, prompt)
}

// BuildEngineFromConfig is a small helper for cmd wiring.
func BuildEngineFromConfig(
	embedClient *embed.Client,
	llmClient *llm.Client,
	q *vectorstore.Qdrant,
	g *graph.Store,
	embedModel, genModel, collection string,
) *Engine {
	return &Engine{
		Embed:      embedClient,
		LLM:        llmClient,
		Qdrant:     q,
		Graph:      g,
		EmbedModel: embedModel,
		GenModel:   genModel,
		Collection: collection,
		TopK:       8,
	}
}
