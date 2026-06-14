// Package embedding configures the embedding model name (embed.* EMbed_MODEL mapping).
package embedding

// Config holds the embedding model identifier passed to Ollama.
type Config struct {
	Model string `default:"embeddinggemma"`
}
