package rag

import (
	"strings"
	"testing"
)

func TestSanitizeBotAnswer(t *testing.T) {
	const dump = `Вот анализ предоставленного текста:

SUBJ-MDK, INVOLVES — Datenschutz (UC-11): Указывает на процесс.
DOC:processes/README.md: Описывает процессы.

Использованные идентификаторы (SUBJ-, UC-):

UC-11
SUBJ-BETRIEBSPRUEFUNG
`

	out := sanitizeBotAnswer(dump)
	if strings.Contains(out, "SUBJ-") || strings.Contains(out, "DOC:") || strings.Contains(out, "INVOLVES") {
		t.Fatalf("expected dump removed, got:\n%s", out)
	}
	if strings.Contains(strings.ToLower(out), "использованные") {
		t.Fatalf("expected ID section removed: %q", out)
	}
}

func TestSanitizeBotAnswer_KeepsShortUC(t *testing.T) {
	in := `UC-01 — кейс про приём нового клиента по базе.`
	out := sanitizeBotAnswer(in)
	if out != in {
		t.Fatalf("expected to keep short answer, got %q", out)
	}
}
