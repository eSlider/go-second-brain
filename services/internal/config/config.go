// Package config loads environment variables for ingestor and bot. No globals.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Config holds shared settings for stores, Ollama, and optional Matrix.
type Config struct {
	DocsRoot string

	Neo4jURI      string
	Neo4jUser     string
	Neo4jPassword string

	QdrantURL        string
	QdrantCollection string

	OllamaURL   string
	EmbedModel  string
	GenModel    string
	HTTPTimeout time.Duration

	MatrixHomeserver string
	MatrixUser       string
	MatrixPassword   string
	MatrixDebug      bool

	BotCommandPrefix string
	BotDBPath        string
}

// Load reads configuration from the environment with defaults suitable for docker-compose.
func Load() (Config, error) {
	c := Config{
		DocsRoot:         getEnv("DOCS_ROOT", "docs/project"),
		Neo4jURI:         os.Getenv("NEO4J_URI"),
		Neo4jUser:        getEnv("NEO4J_USER", "neo4j"),
		Neo4jPassword:    os.Getenv("NEO4J_PASSWORD"),
		QdrantURL:        os.Getenv("QDRANT_URL"),
		QdrantCollection: getEnv("QDRANT_COLLECTION", "edelweiss"),
		OllamaURL:        strings.TrimRight(os.Getenv("OLLAMA_URL"), "/"),
		EmbedModel:       getEnv("EMBED_MODEL", "nomic-embed-text"),
		GenModel:         getEnv("GEN_MODEL", "gemma3:1b"),
		HTTPTimeout:      durationEnv("HTTP_TIMEOUT", 120*time.Second),
		MatrixHomeserver: strings.TrimRight(os.Getenv("MATRIX_API_URL"), "/"),
		MatrixUser:       firstNonEmpty(os.Getenv("MATRIX_USER"), os.Getenv("MATRIX_API_USER")),
		MatrixPassword:   firstNonEmpty(os.Getenv("MATRIX_PASSWORD"), os.Getenv("MATRIX_API_PASS")),
		MatrixDebug:      os.Getenv("MATRIX_DEBUG") == "true",
		BotCommandPrefix: getEnv("BOT_COMMAND_PREFIX", "!edel"),
		BotDBPath:        getEnv("MATRIX_BOT_DB", "/data/matrix-bot.db"),
	}

	if c.Neo4jURI == "" {
		return Config{}, fmt.Errorf("config: NEO4J_URI is required")
	}
	if c.Neo4jPassword == "" {
		return Config{}, fmt.Errorf("config: NEO4J_PASSWORD is required")
	}
	if c.QdrantURL == "" {
		return Config{}, fmt.Errorf("config: QDRANT_URL is required")
	}
	if c.OllamaURL == "" {
		return Config{}, fmt.Errorf("config: OLLAMA_URL is required")
	}
	return c, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func durationEnv(key string, def time.Duration) time.Duration {
	s := os.Getenv(key)
	if s == "" {
		return def
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return def
	}
	return d
}

// IngestorOnly validates fields needed for ingestion (no Matrix).
func (c Config) IngestorOnly() error {
	return nil
}

// Bot validates Matrix + shared fields.
func (c Config) Bot() error {
	if c.MatrixHomeserver == "" {
		return fmt.Errorf("config: MATRIX_API_URL is required for bot")
	}
	if c.MatrixUser == "" || c.MatrixPassword == "" {
		return fmt.Errorf("config: MATRIX_USER and MATRIX_PASSWORD (or MATRIX_API_*) are required for bot")
	}
	return nil
}
