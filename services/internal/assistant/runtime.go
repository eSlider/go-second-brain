package assistant

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/eSlider/go-second-brain/services/internal/config"
	"github.com/eSlider/go-second-brain/services/pkg/cartesia"
	"github.com/eSlider/go-second-brain/services/pkg/inworld"
)

type Runtime struct {
	cfg  *config.Config
	log  *slog.Logger
	perf *PerfLogger
}

func NewRuntime(cfg *config.Config, log *slog.Logger, perf *PerfLogger) *Runtime {
	return &Runtime{cfg: cfg, log: log, perf: perf}
}

func (r *Runtime) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	store, err := NewMediaStore(r.cfg.Assistant.TTS.Dir)
	if err != nil {
		return err
	}
	recStore, err := NewMediaStore(r.cfg.Assistant.Audio.Rec.Dir)
	if err != nil {
		return err
	}
	sttStore, err := NewMediaStore(r.cfg.Assistant.STT.Dir)
	if err != nil {
		return err
	}
	opusSaver := NewAsyncOpusSaver(2, 16)
	defer opusSaver.Close()
	capture, err := StartCapture(ctx, r.cfg.Assistant.Audio.Sample.Rate)
	if err != nil {
		return err
	}
	defer func() {
		_ = capture.Close()
	}()
	playback, err := StartPlayback(r.cfg.Assistant.Audio.Sample.Rate)
	if err != nil {
		return err
	}
	defer func() {
		_ = playback.Close()
	}()
	stt, err := inworld.New(ctx, &r.cfg.Inworld)
	if err != nil {
		return err
	}
	defer func() {
		_ = stt.Close()
	}()
	tts, err := cartesia.New(ctx, &r.cfg.Cartesia)
	if err != nil {
		return err
	}
	defer func() {
		_ = tts.Close()
	}()

	micChunks := make(chan []byte, 64)
	finals := make(chan inworld.Event, 16)
	errCh := make(chan error, 4)
	var wg sync.WaitGroup
	var ttsMu sync.Mutex
	var micRaw bytes.Buffer

	wg.Go(func() {
		if err := capture.StreamChunks(ctx, r.cfg.Assistant.Chunk.MS, r.cfg.Assistant.Audio.Sample.Rate, micChunks); err != nil && ctx.Err() == nil {
			errCh <- err
		}
		close(micChunks)
	})

	wg.Go(func() {
		const (
			voiceThreshold = 0.012 // tune for conversational mic noise floor
			pauseToCutMS   = 700
		)
		speaking := false
		silenceMS := 0
		for chunk := range micChunks {
			micRaw.Write(chunk)
			_ = r.perf.Event("mic_chunk_captured", map[string]any{"bytes": len(chunk)})
			if err := stt.SendPCMChunk(ctx, chunk); err != nil {
				errCh <- err
				return
			}
			// Pause-aware turn detection: cut utterance on conversational silence.
			level := rmsLevelPCM16(chunk)
			chunkMS := r.cfg.Assistant.Chunk.MS
			if chunkMS <= 0 {
				chunkMS = 100
			}
			if level >= voiceThreshold {
				speaking = true
				silenceMS = 0
				continue
			}
			if !speaking {
				continue
			}
			silenceMS += chunkMS
			if silenceMS >= pauseToCutMS {
				if err := stt.SendEndTurn(ctx); err != nil && ctx.Err() == nil {
					errCh <- err
					return
				}
				_ = r.perf.Event("stt_end_turn_sent", map[string]any{"silence_ms": silenceMS})
				speaking = false
				silenceMS = 0
			}
		}
	})

	wg.Go(func() {
		defer close(finals)
		for {
			ev, err := stt.ReadEvent(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				errCh <- err
				return
			}
			if ev.Text == "" {
				continue
			}
			if ev.IsFinal {
				_ = r.perf.Event("stt_final_received", map[string]any{"text_len": len(ev.Text)})
				r.log.Info("assistant final transcript", slog.String("text", ev.Text))
				select {
				case <-ctx.Done():
					return
				case finals <- ev:
				}
				continue
			}
			_ = r.perf.Event("stt_partial_received", map[string]any{"text_len": len(ev.Text)})
			r.log.Info("assistant partial transcript", slog.String("text", ev.Text))
		}
	})

	wg.Go(func() {
		for ev := range finals {
			sttPath := sttStore.NewFilePath("txt")
			if err := sttStore.WriteAll(sttPath, []byte(ev.Text+"\n")); err != nil {
				errCh <- err
				return
			}
			ttsMu.Lock()
			finalReceivedAt := time.Now()
			audioChunks := make(chan []byte, 32)
			ttsErr := make(chan error, 1)
			var ttsRaw bytes.Buffer
			_ = r.perf.Event("tts_request_started", map[string]any{"text_len": len(ev.Text)})
			go func() {
				ttsErr <- tts.Stream(ctx, ev.Text, audioChunks)
				close(audioChunks)
			}()
			firstByteLogged := false
			var firstByteAt time.Time
			var playbackStartAt time.Time
			for chunk := range audioChunks {
				if !firstByteLogged {
					firstByteLogged = true
					firstByteAt = time.Now()
					firstByteDur := firstByteAt.Sub(finalReceivedAt)
					_ = r.perf.Event("tts_first_byte", map[string]any{
						"latency_ms":    firstByteDur.Milliseconds(),
						"latency_human": humanDuration(firstByteDur),
					})
				}
				ttsRaw.Write(chunk)
				if err := playback.WritePCM(chunk); err != nil {
					ttsMu.Unlock()
					errCh <- err
					return
				}
				if playbackStartAt.IsZero() {
					playbackStartAt = time.Now()
				}
			}
			if err := <-ttsErr; err != nil {
				ttsMu.Unlock()
				errCh <- err
				return
			}
			streamDoneAt := time.Now()
			synthesisDur := streamDoneAt.Sub(finalReceivedAt)
			_ = r.perf.Event("tts_stream_completed", map[string]any{
				"latency_ms":    synthesisDur.Milliseconds(),
				"latency_human": humanDuration(synthesisDur),
				"bytes":         ttsRaw.Len(),
			})
			if !playbackStartAt.IsZero() {
				playbackStartDur := playbackStartAt.Sub(finalReceivedAt)
				_ = r.perf.Event("playback_started", map[string]any{
					"text_len":              len(ev.Text),
					"latency_from_final_ms": playbackStartDur.Milliseconds(),
					"latency_human":         humanDuration(playbackStartDur),
				})
			}
			_ = r.perf.Event("playback_completed", map[string]any{
				"latency_ms":    synthesisDur.Milliseconds(),
				"latency_human": humanDuration(synthesisDur),
				"bytes":         ttsRaw.Len(),
			})
			_ = r.perf.Event("end_to_end_ms", map[string]any{
				"value":       synthesisDur.Milliseconds(),
				"value_human": humanDuration(synthesisDur),
			})
			ttsFirstByteMS := int64(-1)
			if !firstByteAt.IsZero() {
				ttsFirstByteMS = firstByteAt.Sub(finalReceivedAt).Milliseconds()
			}
			ttsFirstByteHuman := "n/a"
			if ttsFirstByteMS >= 0 {
				ttsFirstByteHuman = humanDuration(time.Duration(ttsFirstByteMS) * time.Millisecond)
			}
			playbackStartMS := int64(-1)
			if !playbackStartAt.IsZero() {
				playbackStartMS = playbackStartAt.Sub(finalReceivedAt).Milliseconds()
			}
			playbackStartHuman := "n/a"
			if playbackStartMS >= 0 {
				playbackStartHuman = humanDuration(time.Duration(playbackStartMS) * time.Millisecond)
			}
			r.log.Info("assistant timing",
				slog.Int("text_len", len(ev.Text)),
				slog.Int64("tts_first_byte_ms", ttsFirstByteMS),
				slog.String("tts_first_byte", ttsFirstByteHuman),
				slog.Int64("tts_synthesis_ms", synthesisDur.Milliseconds()),
				slog.String("tts_synthesis", humanDuration(synthesisDur)),
				slog.Int64("playback_start_ms", playbackStartMS),
				slog.String("playback_start", playbackStartHuman),
				slog.Int64("playback_done_ms", synthesisDur.Milliseconds()),
				slog.String("playback_done", humanDuration(synthesisDur)),
				slog.Int("audio_bytes", ttsRaw.Len()),
			)
			path := store.NewFilePath("opus")
			ok := opusSaver.Enqueue(ttsRaw.Bytes(), r.cfg.Assistant.Audio.Sample.Rate, path)
			if !ok {
				_ = r.perf.Event("tts_opus_enqueue_dropped", map[string]any{"bytes": ttsRaw.Len()})
			} else {
				_ = r.perf.Event("tts_opus_enqueue_ok", map[string]any{"bytes": ttsRaw.Len()})
			}
			ttsMu.Unlock()
		}
	})

	select {
	case <-ctx.Done():
	case err := <-errCh:
		cancel()
		wg.Wait()
		if err != nil {
			return err
		}
	}

	cancel()
	wg.Wait()
	micPath := recStore.NewFilePath("opus")
	ok := opusSaver.Enqueue(micRaw.Bytes(), r.cfg.Assistant.Audio.Sample.Rate, micPath)
	if !ok {
		return fmt.Errorf("assistant runtime: mic opus enqueue dropped")
	}
	return nil
}
