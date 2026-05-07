package assistant

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"git.produktor.io/edelweiss/docs/services/internal/config"
)

type Runtime struct {
	cfg  config.Config
	log  *slog.Logger
	perf *PerfLogger
}

func NewRuntime(cfg config.Config, log *slog.Logger, perf *PerfLogger) *Runtime {
	return &Runtime{cfg: cfg, log: log, perf: perf}
}

func (r *Runtime) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	store, err := NewMediaStore(r.cfg.AssistantTTSDir)
	if err != nil {
		return err
	}
	recStore, err := NewMediaStore(r.cfg.AssistantAudioRecDir)
	if err != nil {
		return err
	}
	sttStore, err := NewMediaStore(r.cfg.AssistantSTTDir)
	if err != nil {
		return err
	}
	opusSaver := NewAsyncOpusSaver(2, 16)
	defer opusSaver.Close()
	capture, err := StartCapture(ctx, r.cfg.AssistantSampleRate)
	if err != nil {
		return err
	}
	defer func() {
		_ = capture.Close()
	}()
	playback, err := StartPlayback(r.cfg.AssistantSampleRate)
	if err != nil {
		return err
	}
	defer func() {
		_ = playback.Close()
	}()
	stt, err := DialInworldSTT(ctx, r.cfg.InworldAPIKey, r.cfg.InworldSTTModel)
	if err != nil {
		return err
	}
	defer func() {
		_ = stt.Close()
	}()
	tts, err := NewCartesiaTTS(r.cfg.CartesiaAPIKey, r.cfg.CartesiaModelID, r.cfg.CartesiaVoiceID, "ru")
	if err != nil {
		return err
	}

	micChunks := make(chan []byte, 64)
	finals := make(chan STTEvent, 16)
	errCh := make(chan error, 4)
	var wg sync.WaitGroup
	var ttsMu sync.Mutex
	var micRaw bytes.Buffer

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := capture.StreamChunks(ctx, r.cfg.AssistantChunkMS, r.cfg.AssistantSampleRate, micChunks); err != nil && ctx.Err() == nil {
			errCh <- err
		}
		close(micChunks)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for chunk := range micChunks {
			micRaw.Write(chunk)
			_ = r.perf.Event("mic_chunk_captured", map[string]any{"bytes": len(chunk)})
			if err := stt.SendPCMChunk(ctx, chunk); err != nil {
				errCh <- err
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := stt.SendEndTurn(ctx); err != nil && ctx.Err() == nil {
					errCh <- err
					return
				}
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
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
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for ev := range finals {
			sttPath := sttStore.NewFilePath("txt")
			if err := sttStore.WriteAll(sttPath, []byte(ev.Text+"\n")); err != nil {
				errCh <- err
				return
			}
			ttsMu.Lock()
			start := time.Now()
			audioChunks := make(chan []byte, 32)
			ttsErr := make(chan error, 1)
			var ttsRaw bytes.Buffer
			go func() {
				ttsErr <- tts.Stream(ctx, ev.Text, audioChunks)
				close(audioChunks)
			}()
			firstByteLogged := false
			for chunk := range audioChunks {
				if !firstByteLogged {
					firstByteLogged = true
					_ = r.perf.Event("tts_first_byte", map[string]any{"latency_ms": time.Since(start).Milliseconds()})
				}
				ttsRaw.Write(chunk)
				if err := playback.WritePCM(chunk); err != nil {
					ttsMu.Unlock()
					errCh <- err
					return
				}
			}
			if err := <-ttsErr; err != nil {
				ttsMu.Unlock()
				errCh <- err
				return
			}
			_ = r.perf.Event("playback_started", map[string]any{"text_len": len(ev.Text)})
			_ = r.perf.Event("end_to_end_ms", map[string]any{"value": time.Since(start).Milliseconds()})
			path := store.NewFilePath("opus")
			ok := opusSaver.Enqueue(ttsRaw.Bytes(), r.cfg.AssistantSampleRate, path)
			if !ok {
				_ = r.perf.Event("tts_opus_enqueue_dropped", map[string]any{"bytes": ttsRaw.Len()})
			} else {
				_ = r.perf.Event("tts_opus_enqueue_ok", map[string]any{"bytes": ttsRaw.Len()})
			}
			ttsMu.Unlock()
		}
	}()

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
	ok := opusSaver.Enqueue(micRaw.Bytes(), r.cfg.AssistantSampleRate, micPath)
	if !ok {
		return fmt.Errorf("assistant runtime: mic opus enqueue dropped")
	}
	return nil
}
