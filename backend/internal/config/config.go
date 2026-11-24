package config

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
)

const (
	defaultAddress        = "127.0.0.1:8080"
	defaultAllowedOrigins = "http://localhost:5173,http://127.0.0.1:5173,http://localhost:4173,http://127.0.0.1:4173"
	minimumTokenLength    = 32
)

type Config struct {
	Address        string
	AllowedOrigins []string
	DatabasePath   string
	WriteToken     string
}

func Load(lookup func(string) string) (Config, error) {
	cfg := Config{
		Address:      valueOr(lookup("HTTP_ADDR"), defaultAddress),
		DatabasePath: strings.TrimSpace(lookup("DATABASE_PATH")),
		WriteToken:   lookup("API_WRITE_TOKEN"),
	}

	origins, err := parseOrigins(valueOr(lookup("ALLOWED_ORIGINS"), defaultAllowedOrigins))
	if err != nil {
		return Config{}, err
	}
	cfg.AllowedOrigins = origins

	if cfg.WriteToken != "" && len(cfg.WriteToken) < minimumTokenLength {
		return Config{}, fmt.Errorf("API_WRITE_TOKEN must be at least %d characters", minimumTokenLength)
	}
	loopback, err := isLoopback(cfg.Address)
	if err != nil {
		return Config{}, err
	}
	if !loopback && cfg.WriteToken == "" {
		return Config{}, errors.New("API_WRITE_TOKEN is required when HTTP_ADDR is not a loopback address")
	}
	return cfg, nil
}

func valueOr(value, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return fallback
}

func parseOrigins(raw string) ([]string, error) {
	var origins []string
	for _, entry := range strings.Split(raw, ",") {
		origin := strings.TrimSpace(entry)
		if origin == "" {
			continue
		}
		parsed, err := url.Parse(origin)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" ||
			parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil ||
			strings.Contains(origin, "*") {
			return nil, fmt.Errorf("ALLOWED_ORIGINS contains an invalid origin: %q", origin)
		}
		origins = append(origins, origin)
	}
	return origins, nil
}

func isLoopback(address string) (bool, error) {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return false, fmt.Errorf("HTTP_ADDR is invalid: %w", err)
	}
	if host == "localhost" {
		return true, nil
	}
	parsed, err := netip.ParseAddr(host)
	if err != nil {
		return false, nil
	}
	return parsed.IsLoopback(), nil
}
