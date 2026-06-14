// Package httpclient groups shared outbound HTTP timeouts (HTTP_TIMEOUT mapping).
package httpclient

import "time"

// Config holds default timeout for integrations that compose httpjson clients downstream.
type Config struct {
	Timeout time.Duration `default:"120s"`
}
