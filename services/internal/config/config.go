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

	InworldAPIKey        string
	InworldSTTModel      string
	CartesiaAPIKey       string
	CartesiaModelID      string
	CartesiaVoiceID      string
	AssistantSampleRate  int
	AssistantChunkMS     int
	AssistantTTSDir      string
	AssistantAudioRecDir string
	AssistantSTTDir      string
	AssistantPerfLog     string
}

// Load reads configuration from the environment with defaults suitable for docker-compose.
func Load() (Config, error) {
	c := Config{
		DocsRoot:             getEnv("DOCS_ROOT", "docs/project"),
		Neo4jURI:             os.Getenv("NEO4J_URI"),
		Neo4jUser:            getEnv("NEO4J_USER", "neo4j"),
		Neo4jPassword:        os.Getenv("NEO4J_PASSWORD"),
		QdrantURL:            os.Getenv("QDRANT_URL"),
		QdrantCollection:     getEnv("QDRANT_COLLECTION", "edelweiss"),
		OllamaURL:            strings.TrimRight(os.Getenv("OLLAMA_URL"), "/"),
		EmbedModel:           getEnv("EMBED_MODEL", "embeddinggemma"),
		GenModel:             getEnv("GEN_MODEL", "cajina/gemma4_e2b-q4_k_s:v01"),
		HTTPTimeout:          durationEnv("HTTP_TIMEOUT", 120*time.Second),
		MatrixHomeserver:     strings.TrimRight(os.Getenv("MATRIX_API_URL"), "/"),
		MatrixUser:           firstNonEmpty(os.Getenv("MATRIX_USER"), os.Getenv("MATRIX_API_USER")),
		MatrixPassword:       firstNonEmpty(os.Getenv("MATRIX_PASSWORD"), os.Getenv("MATRIX_API_PASS")),
		MatrixDebug:          os.Getenv("MATRIX_DEBUG") == "true",
		BotCommandPrefix:     getEnv("BOT_COMMAND_PREFIX", "!edel"),
		BotDBPath:            getEnv("MATRIX_BOT_DB", "/data/matrix-bot.db"),
		InworldAPIKey:        os.Getenv("INWORLD_API_KEY"),
		InworldSTTModel:      getEnv("INWORLD_STT_MODEL", "base"),
		CartesiaAPIKey:       os.Getenv("CARTESIA_API_KEY"),
		CartesiaModelID:      getEnv("CARTESIA_MODEL_ID", "sonic-3.5"),
		CartesiaVoiceID:      os.Getenv("CARTESIA_VOICE_ID"),
		AssistantSampleRate:  intEnv("ASSISTANT_AUDIO_SAMPLE_RATE", 16000),
		AssistantChunkMS:     intEnv("ASSISTANT_CHUNK_MS", 100),
		AssistantTTSDir:      getEnv("ASSISTANT_TTS_DIR", "var/tss"),
		AssistantAudioRecDir: getEnv("ASSISTANT_AUDIO_REC_DIR", "var/audio-rec"),
		AssistantSTTDir:      getEnv("ASSISTANT_STT_DIR", "var/stt"),
		AssistantPerfLog:     getEnv("ASSISTANT_PERF_LOG", "var/logs/performance.jsonl"),
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

func intEnv(key string, def int) int {
	s := os.Getenv(key)
	if s == "" {
		return def
	}
	var out int
	_, err := fmt.Sscanf(s, "%d", &out)
	if err != nil || out <= 0 {
		return def
	}
	return out
}

// IngestorOnly validates fields needed for ingestion (no Matrix).
func (c Config) IngestorOnly() error {
	if c.Neo4jURI == "" {
		return fmt.Errorf("config: NEO4J_URI is required")
	}
	if c.Neo4jPassword == "" {
		return fmt.Errorf("config: NEO4J_PASSWORD is required")
	}
	if c.QdrantURL == "" {
		return fmt.Errorf("config: QDRANT_URL is required")
	}
	if c.OllamaURL == "" {
		return fmt.Errorf("config: OLLAMA_URL is required")
	}
	return nil
}

// Bot validates Matrix + shared fields.
func (c Config) Bot() error {
	if err := c.IngestorOnly(); err != nil {
		return err
	}
	if c.MatrixHomeserver == "" {
		return fmt.Errorf("config: MATRIX_API_URL is required for bot")
	}
	if c.MatrixUser == "" || c.MatrixPassword == "" {
		return fmt.Errorf("config: MATRIX_USER and MATRIX_PASSWORD (or MATRIX_API_*) are required for bot")
	}
	return nil
}

// Assistant validates STT/TTS + audio settings.
func (c Config) Assistant() error {
	if c.InworldAPIKey == "" {
		return fmt.Errorf("config: INWORLD_API_KEY is required for assistant")
	}
	if c.CartesiaAPIKey == "" {
		return fmt.Errorf("config: CARTESIA_API_KEY is required for assistant")
	}
	if c.CartesiaVoiceID == "" {
		return fmt.Errorf("config: CARTESIA_VOICE_ID is required for assistant")
	}
	if c.AssistantSampleRate <= 0 {
		return fmt.Errorf("config: ASSISTANT_AUDIO_SAMPLE_RATE must be > 0")
	}
	if c.AssistantChunkMS <= 0 {
		return fmt.Errorf("config: ASSISTANT_CHUNK_MS must be > 0")
	}
	return nil
}
