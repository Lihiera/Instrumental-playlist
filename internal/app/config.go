package app

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	envHTTPAddr               = "HTTP_ADDR"
	envSpotifyClientID        = "SPOTIFY_CLIENT_ID"
	envSpotifyClientSecret    = "SPOTIFY_CLIENT_SECRET"
	envSpotifyRedirectURI     = "SPOTIFY_REDIRECT_URI"
	envSpotifyBaseURL         = "SPOTIFY_BASE_URL"
	envSpotifyAccountsBaseURL = "SPOTIFY_ACCOUNTS_BASE_URL"
)

type Config struct {
	HTTPAddr               string
	SpotifyClientID        string
	SpotifyClientSecret    string
	SpotifyRedirectURI     string
	SpotifyBaseURL         string
	SpotifyAccountsBaseURL string
}

type PublicConfig struct {
	HTTPAddr                      string `json:"http_addr"`
	SpotifyClientIDConfigured     bool   `json:"spotify_client_id_configured"`
	SpotifyClientSecretConfigured bool   `json:"spotify_client_secret_configured"`
	SpotifyRedirectURI            string `json:"spotify_redirect_uri"`
	SpotifyBaseURL                string `json:"spotify_base_url"`
	SpotifyAccountsBaseURL        string `json:"spotify_accounts_base_url"`
}

type configOptions struct {
	Environ    []string
	DotenvPath string
}

// LoadConfig builds the runtime configuration from defaults, .env values, and process environment overrides.
func LoadConfig(opts configOptions) (Config, error) {
	cfg := Config{
		HTTPAddr:               ":8080",
		SpotifyBaseURL:         "https://api.spotify.com",
		SpotifyAccountsBaseURL: "https://accounts.spotify.com",
	}

	env, err := loadDotenv(opts.DotenvPath)
	if err != nil {
		return Config{}, err
	}
	mergeEnv(env, envMap(opts.Environ))

	if err := applyEnv(&cfg, env); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Public returns a redacted configuration view that is safe to expose through the API.
func (cfg Config) Public() PublicConfig {
	return PublicConfig{
		HTTPAddr:                      cfg.HTTPAddr,
		SpotifyClientIDConfigured:     strings.TrimSpace(cfg.SpotifyClientID) != "",
		SpotifyClientSecretConfigured: strings.TrimSpace(cfg.SpotifyClientSecret) != "",
		SpotifyRedirectURI:            cfg.SpotifyRedirectURI,
		SpotifyBaseURL:                cfg.SpotifyBaseURL,
		SpotifyAccountsBaseURL:        cfg.SpotifyAccountsBaseURL,
	}
}

// loadDotenv reads a KEY=VALUE .env file into a map and treats a missing file as empty configuration.
func loadDotenv(path string) (map[string]string, error) {
	if strings.TrimSpace(path) == "" {
		return map[string]string{}, nil
	}

	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read .env file %q: %w", path, err)
	}

	env := map[string]string{}
	for lineNo, raw := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("parse .env file %q line %d: expected KEY=VALUE", path, lineNo+1)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("parse .env file %q line %d: empty key", path, lineNo+1)
		}
		env[key] = trimEnvValue(value)
	}

	return env, nil
}

// trimEnvValue removes surrounding whitespace and one layer of matching quotes from an environment value.
func trimEnvValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}

// mergeEnv copies all values from src into dst, overwriting existing keys.
func mergeEnv(dst, src map[string]string) {
	for key, value := range src {
		dst[key] = value
	}
}

// applyEnv applies supported environment variables to cfg and validates typed values.
func applyEnv(cfg *Config, env map[string]string) error {
	if value := strings.TrimSpace(env[envHTTPAddr]); value != "" {
		cfg.HTTPAddr = value
	}
	if value := strings.TrimSpace(env[envSpotifyClientID]); value != "" {
		cfg.SpotifyClientID = value
	}
	if value := strings.TrimSpace(env[envSpotifyClientSecret]); value != "" {
		cfg.SpotifyClientSecret = value
	}
	if value := strings.TrimSpace(env[envSpotifyRedirectURI]); value != "" {
		cfg.SpotifyRedirectURI = value
	}
	if value := strings.TrimSpace(env[envSpotifyBaseURL]); value != "" {
		cfg.SpotifyBaseURL = value
	}
	if value := strings.TrimSpace(env[envSpotifyAccountsBaseURL]); value != "" {
		cfg.SpotifyAccountsBaseURL = value
	}

	return nil
}

// envMap converts os.Environ-style KEY=VALUE entries into a lookup map.
func envMap(environ []string) map[string]string {
	env := make(map[string]string, len(environ))
	for _, entry := range environ {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			env[key] = value
		}
	}
	return env
}
