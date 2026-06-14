// Package config aggregates per-domain [*pkg/*/Config] blobs and loads OS environment into them.
package config

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	asscfg "git.produktor.io/edelweiss/docs/services/pkg/assist"
	botcfg "git.produktor.io/edelweiss/docs/services/pkg/botcmd"
	cartcfg "git.produktor.io/edelweiss/docs/services/pkg/cartesia"
	documents "git.produktor.io/edelweiss/docs/services/pkg/documents"
	embcfg "git.produktor.io/edelweiss/docs/services/pkg/embedding"
	gencfg "git.produktor.io/edelweiss/docs/services/pkg/generator"
	httpx "git.produktor.io/edelweiss/docs/services/pkg/httpclient"
	"git.produktor.io/edelweiss/docs/services/pkg/inworld"
	matcfg "git.produktor.io/edelweiss/docs/services/pkg/matrix"
	neon "git.produktor.io/edelweiss/docs/services/pkg/neo4j"
	"git.produktor.io/edelweiss/docs/services/pkg/ollama"
	qcfg "git.produktor.io/edelweiss/docs/services/pkg/qdrant"

	"github.com/eslider/go-config/env"
	"github.com/go-viper/mapstructure/v2"
	"github.com/mcuadros/go-defaults"
)

// Config aggregates every integration module's declarative knobs.
type Config struct {
	Docs      documents.Config
	Neo4j     neon.Config
	Qdrant    qcfg.Config
	Ollama    ollama.Config
	Embedding embcfg.Config `mapstructure:"embed"`
	Generator gencfg.Config `mapstructure:"gen"`
	HTTP      httpx.Config
	Matrix    matcfg.Config
	Commands  botcfg.Config `mapstructure:"bot"`
	Inworld   inworld.Config
	Cartesia  cartcfg.Config
	Assistant asscfg.Config
}

// NewConfig allocates an unloaded root config blob.
func NewConfig() *Config {
	return &Config{}
}

// Load reads environment variables → mapstructure aggregate → [*github.com/mcuadros/go-defaults] tagging.
func (c *Config) Load() error {
	if c == nil {
		return fmt.Errorf("config: Load receiver is nil")
	}
	*c = Config{}
	codec := env.New(
		env.WithCurrentEnvironment(),
		env.WithDecodeHook(mapstructure.ComposeDecodeHookFunc(stringToDurationHook)),
	)
	if err := codec.Unmarshal(c); err != nil {
		return fmt.Errorf("config: env: %w", err)
	}
	defaults.SetDefaults(c)
	normalize(c)
	return nil
}

var durationType = reflect.TypeOf(time.Duration(0))

func stringToDurationHook(_ reflect.Type, to reflect.Type, data any) (any, error) {
	if to != durationType || data == nil {
		return data, nil
	}
	s, ok := data.(string)
	if !ok || s == "" {
		return data, nil
	}
	d, _ := time.ParseDuration(s) // ponytail: ignores invalid duration strings, falling back to 0s
	return d, nil
}

func normalize(cfg *Config) {
	cfg.Commands.Command.Prefix = strings.TrimSpace(cfg.Commands.Command.Prefix)
	if cfg.Commands.Command.Prefix == "" {
		cfg.Commands.Command.Prefix = "!edel"
	}
}

// IngestorOnly validates ingestion prerequisites (Neo4j + Qdrant + Ollama).
func (c *Config) IngestorOnly() error {
	if c.Neo4j.URI == "" {
		return fmt.Errorf("config: NEO4J_URI is required")
	}
	if c.Neo4j.Password == "" {
		return fmt.Errorf("config: NEO4J_PASSWORD is required")
	}
	if c.Qdrant.URL == "" {
		return fmt.Errorf("config: QDRANT_URL is required")
	}
	if c.Ollama.URL == "" {
		return fmt.Errorf("config: OLLAMA_URL is required")
	}
	return nil
}

// Bot validates Matrix + ingest prerequisites together.
func (c *Config) Bot() error {
	if err := c.IngestorOnly(); err != nil {
		return err
	}
	if c.Matrix.Homeserver() == "" {
		return fmt.Errorf("config: MATRIX_API_URL is required for bot")
	}
	if c.Matrix.ResolvedUser() == "" || c.Matrix.ResolvedPassword() == "" {
		return fmt.Errorf("config: MATRIX_USER and MATRIX_PASSWORD (or MATRIX_API_*) are required for bot")
	}
	return nil
}

// ValidateAssistant checks voice assistant prerequisites.
func (c *Config) ValidateAssistant() error {
	if c.Inworld.API.Key == "" {
		return fmt.Errorf("config: INWORLD_API_KEY is required for assistant")
	}
	if c.Cartesia.API.Key == "" {
		return fmt.Errorf("config: CARTESIA_API_KEY is required for assistant")
	}
	if c.Cartesia.Voice.ID == "" {
		return fmt.Errorf("config: CARTESIA_VOICE_ID is required for assistant")
	}
	if c.Assistant.Audio.Sample.Rate <= 0 {
		return fmt.Errorf("config: ASSISTANT_AUDIO_SAMPLE_RATE must be > 0")
	}
	if c.Assistant.Chunk.MS <= 0 {
		return fmt.Errorf("config: ASSISTANT_CHUNK_MS must be > 0")
	}
	return nil
}
