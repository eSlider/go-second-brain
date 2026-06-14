// Package ollama is an HTTP client for the Ollama API (embeddings + generate).
package ollama

import (
	"time"
)

// Config configures the Ollama HTTP endpoint (OLLAMA_URL + HTTP_TIMEOUT).
type Config struct {
	URL     string
	Timeout time.Duration `default:"120s"`
}

//func GetClient(cfg *Config) (ctx context.Context) (client *Client, err error) {
//	return New(ctx, cfg)
//}
