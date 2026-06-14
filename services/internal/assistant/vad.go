package assistant

import "encoding/binary"

// rmsLevelPCM16 returns normalized RMS [0..1] for mono PCM16LE.
func rmsLevelPCM16(pcm []byte) float64 {
	if len(pcm) < 2 {
		return 0
	}
	var sum float64
	samples := 0
	for i := 0; i+1 < len(pcm); i += 2 {
		v := int16(binary.LittleEndian.Uint16(pcm[i : i+2]))
		n := float64(v) / 32768.0
		sum += n * n
		samples++
	}
	if samples == 0 {
		return 0
	}
	mean := sum / float64(samples)
	if mean <= 0 {
		return 0
	}
	// sqrt without importing math package for tiny hot path.
	z := mean
	for range 6 {
		z -= (z*z - mean) / (2 * z)
	}
	if z < 0 {
		return 0
	}
	return z
}
