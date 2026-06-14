// Package cartesia is a WebSocket TTS client for the Cartesia API.
package cartesia

// Config holds voice synthesis parameters mapped from CARTESIA_* env keys.
type Config struct {
	API struct {
		Key string // API token (often called API key).
	}
	Model struct {
		ID string `default:"sonic-3.5"`
	}
	Voice struct {
		ID string
	}
	Language   string `default:"ru"`
	SampleRate int    `default:"16000"` // PCM output sample rate sent to Cartesia output_format.
}
