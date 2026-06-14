// Package documents configures documentation corpus roots on disk (DOCS_ROOT).
package documents

// Config selects the Markdown/docs tree scanned by ingestion.
type Config struct {
	Root string `default:"docs/project"`
}
