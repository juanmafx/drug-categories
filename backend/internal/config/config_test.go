package config

import (
	"strings"
	"testing"
)

var validToken = strings.Repeat("v", 32)

func lookupFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestLoadDefaultsToLoopback(t *testing.T) {
	cfg, err := Load(lookupFrom(nil))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Address != "127.0.0.1:8080" || len(cfg.AllowedOrigins) != 4 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadRequiresTokenOffLoopback(t *testing.T) {
	for _, address := range []string{":8080", "0.0.0.0:8080", "[::]:8080", "192.168.1.10:8080", "example.test:8080"} {
		if _, err := Load(lookupFrom(map[string]string{"HTTP_ADDR": address})); err == nil {
			t.Fatalf("expected error for %s", address)
		}
		if _, err := Load(lookupFrom(map[string]string{"HTTP_ADDR": address, "API_WRITE_TOKEN": validToken})); err != nil {
			t.Fatalf("unexpected error for %s: %v", address, err)
		}
	}
}

func TestLoadAcceptsLoopbackVariants(t *testing.T) {
	for _, address := range []string{"localhost:9000", "127.0.0.1:9000", "[::1]:9000"} {
		if _, err := Load(lookupFrom(map[string]string{"HTTP_ADDR": address})); err != nil {
			t.Fatalf("unexpected error for %s: %v", address, err)
		}
	}
}

func TestLoadRejectsShortToken(t *testing.T) {
	_, err := Load(lookupFrom(map[string]string{"API_WRITE_TOKEN": "short"}))
	if err == nil || strings.Contains(err.Error(), "short") {
		t.Fatalf("got %v", err)
	}
}

func TestLoadRejectsMalformedAddress(t *testing.T) {
	if _, err := Load(lookupFrom(map[string]string{"HTTP_ADDR": "not-an-address"})); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadRejectsInvalidOrigins(t *testing.T) {
	for _, origin := range []string{"*", "https://*.example.test", "ftp://example.test", "https://example.test/path", "example.test", "https://user@example.test"} {
		if _, err := Load(lookupFrom(map[string]string{"ALLOWED_ORIGINS": origin})); err == nil {
			t.Fatalf("expected error for %s", origin)
		}
	}
}

func TestLoadParsesOriginList(t *testing.T) {
	cfg, err := Load(lookupFrom(map[string]string{"ALLOWED_ORIGINS": " https://a.example.test , https://b.example.test:8443,"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.AllowedOrigins) != 2 || cfg.AllowedOrigins[1] != "https://b.example.test:8443" {
		t.Fatalf("got %v", cfg.AllowedOrigins)
	}
}

func TestLoadRejectsTokenWithWhitespaceOrControlCharacters(t *testing.T) {
	for _, token := range []string{
		strings.Repeat(" ", 32),
		validToken + " ",
		" " + validToken,
		validToken + "\n",
		validToken[:16] + " " + validToken[16:],
		validToken + "\x00",
	} {
		if _, err := Load(lookupFrom(map[string]string{"API_WRITE_TOKEN": token})); err == nil {
			t.Fatalf("expected error for token of length %d", len(token))
		}
	}
}
