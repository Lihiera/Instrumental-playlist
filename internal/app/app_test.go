package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWithoutArgsStartsHTTPServer(t *testing.T) {
	var stdout bytes.Buffer
	var gotAddr string
	var gotHandler http.Handler

	app := &App{
		Stdout:     &stdout,
		Stderr:     &bytes.Buffer{},
		Environ:    []string{envHTTPAddr + "=127.0.0.1:9090"},
		DotenvPath: filepath.Join(t.TempDir(), "missing.env"),
		ListenAndServe: func(addr string, handler http.Handler) error {
			gotAddr = addr
			gotHandler = handler
			return nil
		},
	}

	if err := app.Run(nil); err != nil {
		t.Fatalf("Run(nil) returned error: %v", err)
	}
	if gotAddr != "127.0.0.1:9090" {
		t.Fatalf("addr = %q", gotAddr)
	}
	if gotHandler == nil {
		t.Fatal("handler was nil")
	}
	if !strings.Contains(stdout.String(), "listening on 127.0.0.1:9090") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunHelpPrintsWebAPIUsage(t *testing.T) {
	var stdout bytes.Buffer
	app := &App{Stdout: &stdout, Stderr: &bytes.Buffer{}}

	if err := app.Run([]string{"help"}); err != nil {
		t.Fatalf("Run(help) returned error: %v", err)
	}

	out := stdout.String()
	for _, want := range []string{"Usage:", "GET /health", "APPLE_DEVELOPER_TOKEN"} {
		if !strings.Contains(out, want) {
			t.Fatalf("usage output missing %q:\n%s", want, out)
		}
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	app := &App{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}

	err := app.Run([]string{"config"})
	if err == nil {
		t.Fatal("Run returned nil for an unknown command")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadConfigReadsDotenvAndEnvironmentOverrides(t *testing.T) {
	dotenvPath := filepath.Join(t.TempDir(), ".env")
	writeFile(t, dotenvPath, `
HTTP_ADDR=:8081
APPLE_DEVELOPER_TOKEN=token-from-file
APPLE_STOREFRONT=jp
INSTRUMENTAL_THRESHOLD=0.8
`)

	cfg, err := LoadConfig(configOptions{
		Environ: []string{
			envHTTPAddr + "=127.0.0.1:9090",
			envInstrumentalThreshold + "=0.9",
		},
		DotenvPath: dotenvPath,
	})
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if cfg.HTTPAddr != "127.0.0.1:9090" {
		t.Fatalf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.AppleDeveloperToken != "token-from-file" {
		t.Fatalf("AppleDeveloperToken = %q", cfg.AppleDeveloperToken)
	}
	if cfg.AppleStorefront != "jp" {
		t.Fatalf("AppleStorefront = %q", cfg.AppleStorefront)
	}
	if cfg.InstrumentalThreshold != 0.9 {
		t.Fatalf("InstrumentalThreshold = %v", cfg.InstrumentalThreshold)
	}
}

func TestLoadConfigRejectsInvalidDotenvAndThreshold(t *testing.T) {
	tests := []struct {
		name        string
		dotenv      string
		environ     []string
		wantMessage string
	}{
		{
			name:        "invalid dotenv",
			dotenv:      "BROKEN",
			wantMessage: "expected KEY=VALUE",
		},
		{
			name:        "invalid threshold",
			dotenv:      "INSTRUMENTAL_THRESHOLD=1.5",
			wantMessage: envInstrumentalThreshold,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dotenvPath := filepath.Join(t.TempDir(), ".env")
			writeFile(t, dotenvPath, tt.dotenv)

			_, err := LoadConfig(configOptions{Environ: tt.environ, DotenvPath: dotenvPath})
			if err == nil {
				t.Fatal("LoadConfig returned nil")
			}
			if !strings.Contains(err.Error(), tt.wantMessage) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.wantMessage)
			}
		})
	}
}

func TestHealthEndpoint(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	Handler(Config{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"status":"ok"}` {
		t.Fatalf("body = %s", got)
	}
}

func TestConfigEndpointRedactsDeveloperToken(t *testing.T) {
	cfg := Config{
		HTTPAddr:              ":9090",
		AppleDeveloperToken:   "secret-token",
		AppleStorefront:       "jp",
		InstrumentalThreshold: 0.85,
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/config", nil)

	Handler(cfg).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "secret-token") {
		t.Fatalf("config response leaked token: %s", rec.Body.String())
	}

	var got PublicConfig
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !got.AppleDeveloperTokenConfigured {
		t.Fatal("AppleDeveloperTokenConfigured = false")
	}
	if got.HTTPAddr != ":9090" || got.AppleStorefront != "jp" || got.InstrumentalThreshold != 0.85 {
		t.Fatalf("unexpected config response: %+v", got)
	}
}

func TestUnsupportedMethodsReturnMethodNotAllowed(t *testing.T) {
	for _, path := range []string{"/health", "/v1/config"} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, path, nil)

			Handler(Config{}).ServeHTTP(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf("status = %d", rec.Code)
			}
		})
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
