package assistant

import (
	"fmt"
	"io"
	"os/exec"
)

// Playback writes PCM16 mono audio to the default PipeWire output.
type Playback struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
}

func StartPlayback(sampleRate int) (*Playback, error) {
	if sampleRate <= 0 {
		return nil, fmt.Errorf("assistant playback: sampleRate must be > 0")
	}
	bin, args, err := playbackCommand(sampleRate)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(bin, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("assistant playback: stdin pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("assistant playback: start %s: %w", bin, err)
	}
	return &Playback{cmd: cmd, stdin: stdin}, nil
}

func (p *Playback) WritePCM(chunk []byte) error {
	if len(chunk) == 0 {
		return nil
	}
	_, err := p.stdin.Write(chunk)
	if err != nil {
		return fmt.Errorf("assistant playback: write: %w", err)
	}
	return nil
}

func (p *Playback) Close() error {
	if p == nil {
		return nil
	}
	if p.stdin != nil {
		_ = p.stdin.Close()
	}
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
		_, _ = p.cmd.Process.Wait()
	}
	return nil
}

func playbackCommand(sampleRate int) (string, []string, error) {
	if bin, err := exec.LookPath("pw-play"); err == nil {
		return bin, []string{
			"--raw",
			"--format", "s16",
			"--rate", fmt.Sprintf("%d", sampleRate),
			"--channels", "1",
			"-",
		}, nil
	}
	if bin, err := exec.LookPath("paplay"); err == nil {
		return bin, []string{
			"--raw",
			"--format=s16le",
			fmt.Sprintf("--rate=%d", sampleRate),
			"--channels=1",
			"--stream-name=second-brain-assistant",
		}, nil
	}
	return "", nil, fmt.Errorf("assistant playback: neither pw-cat nor paplay found")
}
