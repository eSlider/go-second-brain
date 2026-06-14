// Package inworld is an STT WebSocket client for the Inworld realtime API.
package inworld

// Config maps INWORLD_* environment keys.
type Config struct {
	API struct {
		Key string
	}
	STT struct {
		Model string `default:"base"`
	}
}
