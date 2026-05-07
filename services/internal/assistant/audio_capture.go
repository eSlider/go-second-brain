package assistant

import (
	"context"
	"fmt"
	"io"
	"os/exec"
)

const bytesPerSamplePCM16 = 2

// Capture reads PCM16 mono audio from the default PipeWire input.
type Capture struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
}

// StartCapture starts microphone capture at the requested sample rate.
func StartCapture(ctx context.Context, sampleRate int) (*Capture, error) {
	if sampleRate <= 0 {
		return nil, fmt.Errorf("assistant capture: sampleRate must be > 0")
	}
	bin, args, err := captureCommand(sampleRate)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("assistant capture: stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("assistant capture: start %s: %w", bin, err)
	}
	return &Capture{cmd: cmd, stdout: stdout}, nil
}

// StreamChunks emits fixed-size PCM chunks for low-latency streaming.
func (c *Capture) StreamChunks(ctx context.Context, chunkMS int, sampleRate int, out chan<- []byte) error {
	if chunkMS <= 0 {
		return fmt.Errorf("assistant capture: chunkMS must be > 0")
	}
	if sampleRate <= 0 {
		return fmt.Errorf("assistant capture: sampleRate must be > 0")
	}
	chunkBytes := sampleRate * bytesPerSamplePCM16 * chunkMS / 1000
	if chunkBytes <= 0 {
		return fmt.Errorf("assistant capture: computed chunk size is 0")
	}
	buf := make([]byte, chunkBytes)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		_, err := io.ReadFull(c.stdout, buf)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return nil
			}
			return fmt.Errorf("assistant capture: read: %w", err)
		}
		chunk := make([]byte, len(buf))
		copy(chunk, buf)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case out <- chunk:
		}
	}
}

func (c *Capture) Close() error {
	if c == nil || c.cmd == nil || c.cmd.Process == nil {
		return nil
	}
	_ = c.cmd.Process.Kill()
	_ = c.stdout.Close()
	_, _ = c.cmd.Process.Wait()
	return nil
}

func captureCommand(sampleRate int) (string, []string, error) {
	if bin, err := exec.LookPath("pw-cat"); err == nil {
		return bin, []string{
			"--record",
			"--format", "s16",
			"--rate", fmt.Sprintf("%d", sampleRate),
			"--channels", "1",
			"-",
		}, nil
	}
	if bin, err := exec.LookPath("parec"); err == nil {
		return bin, []string{
			"--raw",
			"--format=s16le",
			fmt.Sprintf("--rate=%d", sampleRate),
			"--channels=1",
		}, nil
	}
	return "", nil, fmt.Errorf("assistant capture: neither pw-cat nor parec found")
}
