// Package config aggregates per-domain [*pkg/*/Config] blobs and loads YAML + environment into them.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	asscfg "github.com/eSlider/go-second-brain/services/pkg/assist"
	botcfg "github.com/eSlider/go-second-brain/services/pkg/botcmd"
	cartcfg "github.com/eSlider/go-second-brain/services/pkg/cartesia"
	documents "github.com/eSlider/go-second-brain/services/pkg/documents"
	embcfg "github.com/eSlider/go-second-brain/services/pkg/embedding"
	gencfg "github.com/eSlider/go-second-brain/services/pkg/generator"
	httpx "github.com/eSlider/go-second-brain/services/pkg/httpclient"
	"github.com/eSlider/go-second-brain/services/pkg/inworld"
	matcfg "github.com/eSlider/go-second-brain/services/pkg/matrix"
	neon "github.com/eSlider/go-second-brain/services/pkg/neo4j"
	"github.com/eSlider/go-second-brain/services/pkg/ollama"
	qcfg "github.com/eSlider/go-second-brain/services/pkg/qdrant"

	"github.com/eslider/go-config/env"
	"github.com/eslider/go-config/yaml"
	"github.com/go-viper/mapstructure/v2"
	"github.com/mcuadros/go-defaults"
)

// RAGConfig tunes retrieval-augmented generation.
type RAGConfig struct {
	TopK         int    `default:"8"`
	SystemPrompt string `mapstructure:"system_prompt"`
}

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
	RAG       RAGConfig
}

// NewConfig allocates an unloaded root config blob.
func NewConfig() *Config {
	return &Config{}
}

// Load reads config.yaml.example → config.yaml → .env → process environment.
func (c *Config) Load() error {
	if c == nil {
		return fmt.Errorf("config: Load receiver is nil")
	}
	*c = Config{}
	hook := mapstructure.ComposeDecodeHookFunc(stringToDurationHook)
	decodeHook := yaml.WithDecodeHook(hook)

	root := repoRoot()
	yamlOpts := []yaml.Option{decodeHook}
	for _, name := range []string{"config.yaml.example", "config.yaml"} {
		path := filepath.Join(root, name)
		if _, err := os.Stat(path); err == nil {
			yamlOpts = append(yamlOpts, yaml.WithFile(path))
		}
	}
	if err := yaml.New(yamlOpts...).Unmarshal(c); err != nil {
		return fmt.Errorf("config: yaml: %w", err)
	}

	envOpts := []env.Option{
		env.WithDecodeHook(hook),
		env.WithCurrentEnvironment(),
	}
	for _, name := range []string{".env", filepath.Join(root, ".env")} {
		if _, err := os.Stat(name); err == nil {
			envOpts = append([]env.Option{env.WithFile(name)}, envOpts...)
			break
		}
	}
	if err := env.New(envOpts...).Unmarshal(c); err != nil {
		return fmt.Errorf("config: env: %w", err)
	}
	defaults.SetDefaults(c)
	normalize(c)
	return nil
}

func repoRoot() string {
	if p := strings.TrimSpace(os.Getenv("CONFIG_PATH")); p != "" {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return filepath.Dir(p)
		}
		return p
	}
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "config.yaml.example")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return dir
		}
		dir = parent
	}
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
		cfg.Commands.Command.Prefix = "!brain"
	}
	if strings.TrimSpace(cfg.RAG.SystemPrompt) == "" {
		cfg.RAG.SystemPrompt = defaultRAGSystemPrompt()
	}
}

func defaultRAGSystemPrompt() string {
	return strings.Join([]string{
		"You are a knowledge-base assistant.",
		"Answer in Russian, 2–8 sentences, grounded only in provided excerpts.",
		"If the excerpts do not contain the answer, say briefly that it is not in the knowledge base.",
	}, "\n")
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
