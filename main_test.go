package main

import (
	"strings"
	"testing"
	"time"
)

const sampleLog = `2026-09-18T14:00:01Z INF Starting tunnel tunnelID=abc
2026-09-18T14:00:03Z INF +--------------------------------------------------------------------------------------------+
2026-09-18T14:00:03Z INF |  Your quick Tunnel has been created! Visit it at (it may take some time to be reachable):  |
2026-09-18T14:00:03Z INF |  https://diploma-hosting-pda-palmer.trycloudflare.com                                      |
2026-09-18T14:00:03Z INF +--------------------------------------------------------------------------------------------+
2026-09-18T14:00:04Z INF Registered tunnel connection
`

func TestExtractTunnelURLFound(t *testing.T) {
	url, err := extractTunnelURL(strings.NewReader(sampleLog), 5*time.Second)
	if err != nil {
		t.Fatalf("esperaba URL, hubo error: %v", err)
	}
	want := "https://diploma-hosting-pda-palmer.trycloudflare.com"
	if url != want {
		t.Fatalf("url = %q, quería %q", url, want)
	}
}

func TestExtractTunnelURLTimeout(t *testing.T) {
	_, err := extractTunnelURL(strings.NewReader("sin url aqui\n"), 300*time.Millisecond)
	if err == nil {
		t.Fatal("esperaba error por tiempo agotado")
	}
}

func TestTunnelRegexVariants(t *testing.T) {
	cases := map[string]string{
		"visit https://abc-123.trycloudflare.com now": "https://abc-123.trycloudflare.com",
		"https://xyz.trycloudflare.com/":              "https://xyz.trycloudflare.com",
	}
	for in, want := range cases {
		if got := tunnelRe.FindString(in); got != want {
			t.Errorf("in %q: got %q, want %q", in, got, want)
		}
	}
	if got := tunnelRe.FindString("https://example.com"); got != "" {
		t.Errorf("no debía coincidir: %q", got)
	}
}
