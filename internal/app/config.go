package app

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	envHTTPAddr              = "HTTP_ADDR"
	envAppleDeveloperToken   = "APPLE_DEVELOPER_TOKEN"
	envAppleStorefront       = "APPLE_STOREFRONT"
	envInstrumentalThreshold = "INSTRUMENTAL_THRESHOLD"
)

type Config struct {
	HTTPAddr              string
	AppleDeveloperToken   string
	AppleStorefront       string
	InstrumentalThreshold float64
}

type PublicConfig struct {
	HTTPAddr                      string  `json:"http_addr"`
	AppleDeveloperTokenConfigured bool    `json:"apple_developer_token_configured"`
	AppleStorefront               string  `json:"apple_storefront"`
	InstrumentalThreshold         float64 `json:"instrumental_threshold"`
}

type configOptions struct {
	Environ    []string
	DotenvPath string
}

func LoadConfig(opts configOptions) (Config, error) {
	cfg := Config{
		HTTPAddr:              ":8080",
		AppleStorefront:       "us",
		InstrumentalThreshold: 0.75,
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

func (cfg Config) Public() PublicConfig {
	return PublicConfig{
		HTTPAddr:                      cfg.HTTPAddr,
		AppleDeveloperTokenConfigured: strings.TrimSpace(cfg.AppleDeveloperToken) != "",
		AppleStorefront:               cfg.AppleStorefront,
		InstrumentalThreshold:         cfg.InstrumentalThreshold,
	}
}

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

func trimEnvValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}

func mergeEnv(dst, src map[string]string) {
	for key, value := range src {
		dst[key] = value
	}
}

func applyEnv(cfg *Config, env map[string]string) error {
	if value := strings.TrimSpace(env[envHTTPAddr]); value != "" {
		cfg.HTTPAddr = value
	}
	if value := strings.TrimSpace(env[envAppleDeveloperToken]); value != "" {
		cfg.AppleDeveloperToken = value
	}
	if value := strings.TrimSpace(env[envAppleStorefront]); value != "" {
		cfg.AppleStorefront = value
	}

	rawThreshold := strings.TrimSpace(env[envInstrumentalThreshold])
	if rawThreshold == "" {
		return nil
	}
	threshold, err := strconv.ParseFloat(rawThreshold, 64)
	if err != nil || threshold <= 0 || threshold > 1 {
		return fmt.Errorf("%s must be greater than 0 and less than or equal to 1", envInstrumentalThreshold)
	}
	cfg.InstrumentalThreshold = threshold

	return nil
}

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
