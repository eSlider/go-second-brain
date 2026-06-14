package inworld

import (
	"fmt"

	"github.com/go-viper/mapstructure/v2"
)

// VoiceProfileBucket is one bucket in a categorical voice facet.
type VoiceProfileBucket struct {
	Label      string  `json:"label"`
	Confidence float64 `json:"confidence"`
}

// VoiceProfile contains optional categorical facets emitted by STT.
type VoiceProfile struct {
	Age        []VoiceProfileBucket `json:"age"`
	Accent     []VoiceProfileBucket `json:"accent"`
	Emotion    []VoiceProfileBucket `json:"emotion"`
	VocalStyle []VoiceProfileBucket `json:"vocal_style"`
	Pitch      []VoiceProfileBucket `json:"pitch"`
}

// Event is the merged STT transcription event produced from API envelopes.
type Event struct {
	Text         string       `json:"text"`
	IsFinal      bool         `json:"isFinal"`
	VoiceProfile VoiceProfile `json:"voice_profile"`
}

// wireEvent decodes JSON frames from Inworld WebSocket payloads.
type wireEvent struct {
	Message string `json:"message"`
	Code    float64 `json:"code"`

	Result            *wireResult               `json:"result"`
	TranscriptionRoot *wireTranscriptionLine   `json:"transcription"`
	TranscriptRoot    *wireTranscriptBlob      `json:"transcript"`
}

type wireResult struct {
	Transcription     *wireTranscriptionLine `json:"transcription"`
	VoiceProfile      *VoiceProfile          `json:"voiceProfile"`
	VoiceProfileSnake *VoiceProfile          `json:"voice_profile"`
}

type wireTranscriptionLine struct {
	Transcript string `json:"transcript"`
	IsFinal    bool   `json:"isFinal"`
}

type wireTranscriptBlob struct {
	Text    string `json:"text"`
	IsFinal bool   `json:"isFinal"`
}

func (e wireEvent) merged() Event {
	var out Event
	if r := e.Result; r != nil {
		copyTranscriptionInto(&out, r.Transcription)
		if r.VoiceProfile != nil {
			out.VoiceProfile = *r.VoiceProfile
		}
		if r.VoiceProfileSnake != nil {
			out.VoiceProfile = *r.VoiceProfileSnake
		}
	}
	copyTranscriptionInto(&out, e.TranscriptionRoot)
	if b := e.TranscriptRoot; b != nil {
		out.Text = b.Text
		out.IsFinal = b.IsFinal
	}
	return out
}

func copyTranscriptionInto(out *Event, src *wireTranscriptionLine) {
	if src == nil {
		return
	}
	out.Text = src.Transcript
	out.IsFinal = src.IsFinal
}

func decodeWireEvent(raw map[string]any) (wireEvent, error) {
	var ev wireEvent
	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName:              "json",
		WeaklyTypedInput:     true,
		IgnoreUntaggedFields: true,
		Result:               &ev,
	})
	if err != nil {
		return wireEvent{}, fmt.Errorf("inworld stt: mapstructure decoder: %w", err)
	}
	if err := dec.Decode(raw); err != nil {
		return wireEvent{}, fmt.Errorf("inworld stt: decode event: %w", err)
	}
	return ev, nil
}
