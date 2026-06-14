// Package rag orchestrates retrieval and generation.
package rag

import (
	"context"
	"fmt"
	"strings"

	"git.produktor.io/edelweiss/docs/services/internal/graph"
	"git.produktor.io/edelweiss/docs/services/pkg/ollama"
	"git.produktor.io/edelweiss/docs/services/pkg/qdrant"
)

// Engine ties stores and models.
type Engine struct {
	Ollama     *ollama.Client
	Qdrant     *qdrant.Client
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
	system := strings.Join([]string{
		"Ты — ассистент по базе знаний Edelweiss Pflegedienst (служба ухода на дому в Германии).",
		"Правила ответа:",
		"- Язык: русский, 2–8 предложений, разговорный тон, без канцелярита и без списков «анализ».",
		"- Немецкие термины (Pflegegrad, Verordnung, Krankenkasse, Dienstplan и т.д.) не переводи насильно, оставляй в оригинале.",
		"- НЕ перечисляй и НЕ упоминай: коды SUBJ-, UC-, PAIN-, AUTO-, AGENT-, ROAD-, DOC:, пути к файлам, «связи», INVOLVES, ноды графа.",
		"- НЕ пиши заголовков и вступлений: «Вот анализ», «Результаты анализа», «Использованные идентификаторы», «В контексте».",
		"- Если в тексте ниже нет ответа на вопрос — одно короткое предложение: в базе этого нет.",
		"- Ответь только суть по вопросу, без пересказа всего текста.",
	}, "\n")
	prompt := strings.Join([]string{
		"Ответь на вопрос, опираясь только на фрагменты ниже.",
		"",
		"Вопрос:",
		userQuery,
		"",
		"Фрагменты из базы:",
		contextBlock,
	}, "\n")
	raw, err := e.Ollama.GenerateWithSystem(ctx, e.GenModel, system, prompt)
	if err != nil {
		return "", err
	}
	return sanitizeBotAnswer(raw), nil
}

// BuildEngine wires shared Ollama + Qdrant + graph accessors for cmds.
func BuildEngineFromConfig(
	ollamaClient *ollama.Client,
	qdr *qdrant.Client,
	g *graph.Store,
	embedModel, genModel, collection string,
) *Engine {
	return &Engine{
		Ollama:     ollamaClient,
		Qdrant:     qdr,
		Graph:      g,
		EmbedModel: embedModel,
		GenModel:   genModel,
		Collection: collection,
		TopK:       8,
	}
}
