// Package qdrant is a REST client for Qdrant collections and points.
package qdrant

// Config selects the Qdrant HTTP API base URL used by ingest and search.
type Config struct {
	URL string
	// Collection is the default logical collection name (optional for API calls that pass name explicitly).
	Collection string `default:"knowledge"`
}
