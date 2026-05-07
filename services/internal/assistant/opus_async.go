package assistant

import (
	"bytes"
	"fmt"
	"os/exec"
	"sync"
)

type opusJob struct {
	pcm      []byte
	outPath  string
	sampleHz int
}

type AsyncOpusSaver struct {
	jobs chan opusJob
	wg   sync.WaitGroup
}

func NewAsyncOpusSaver(workers int, queue int) *AsyncOpusSaver {
	if workers <= 0 {
		workers = 1
	}
	if queue <= 0 {
		queue = 8
	}
	s := &AsyncOpusSaver{
		jobs: make(chan opusJob, queue),
	}
	for i := 0; i < workers; i++ {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			for job := range s.jobs {
				_ = encodePCMToOpus(job.pcm, job.sampleHz, job.outPath)
			}
		}()
	}
	return s
}

func (s *AsyncOpusSaver) Enqueue(pcm []byte, sampleHz int, outPath string) bool {
	if len(pcm) == 0 {
		return true
	}
	cp := make([]byte, len(pcm))
	copy(cp, pcm)
	select {
	case s.jobs <- opusJob{pcm: cp, sampleHz: sampleHz, outPath: outPath}:
		return true
	default:
		return false
	}
}

func (s *AsyncOpusSaver) Close() {
	if s == nil {
		return
	}
	close(s.jobs)
	s.wg.Wait()
}

func encodePCMToOpus(pcm []byte, sampleHz int, outPath string) error {
	cmd := exec.Command(
		"ffmpeg",
		"-hide_banner", "-loglevel", "error", "-y",
		"-f", "s16le",
		"-ar", fmt.Sprintf("%d", sampleHz),
		"-ac", "1",
		"-i", "pipe:0",
		"-c:a", "libopus",
		"-b:a", "24k",
		"-application", "voip",
		outPath,
	)
	cmd.Stdin = bytes.NewReader(pcm)
	return cmd.Run()
}
