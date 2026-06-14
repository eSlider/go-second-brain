// Package assist configures local assistant audio dirs and pacing (ASSISTANT_* keys).
package assist

// Config maps ASSISTANT_* environment subtree.
type Config struct {
	Audio struct {
		Sample struct {
			Rate int `default:"16000"`
		}
		Rec struct {
			Dir string `default:"var/audio-rec"`
		}
	}
	Chunk struct {
		MS int `default:"100"`
	}
	TTS struct {
		Dir string `default:"var/tss"`
	}
	STT struct {
		Dir string `default:"var/stt"`
	}
	Perf struct {
		Log string `default:"var/logs/performance.jsonl"`
	}
}
