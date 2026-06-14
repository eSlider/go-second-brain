// Package matrix configures Matrix/Synapse connectivity for bots.
package matrix

import "strings"

// Config maps MATRIX_* and matrix.bot.db.
type Config struct {
	API struct {
		URL  string
		User string
		Pass string
	}
	User     string
	Password string
	Debug    bool
	Bot      struct {
		DB string `default:"/data/matrix-bot.db"`
	}
}

// ResolvedUser returns MATRIX_USER if set, otherwise MATRIX_API_USER.
func (c Config) ResolvedUser() string {
	return firstNonEmpty(c.User, c.API.User)
}

// ResolvedPassword returns MATRIX_PASSWORD if set, otherwise MATRIX_API_PASS.
func (c Config) ResolvedPassword() string {
	return firstNonEmpty(c.Password, c.API.Pass)
}

// Homeserver returns MATRIX_API_URL with trimmed trailing slashes.
func (c Config) Homeserver() string {
	return strings.TrimRight(strings.TrimSpace(c.API.URL), "/")
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
