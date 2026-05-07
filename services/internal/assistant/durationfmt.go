package assistant

import (
	"fmt"
	"time"
)

func humanDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	sec := float64(d) / float64(time.Second)
	return fmt.Sprintf("%.1fs", sec)
}

