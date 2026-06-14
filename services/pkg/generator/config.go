// Package generator configures the generation/completion model (gen.* / GEN_MODEL).
package generator

// Config holds generation model identifier for Ollama /api/generate.
type Config struct {
	Model string `default:"cajina/gemma4_e2b-q4_k_s:v01"`
}
