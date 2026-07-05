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

	"github.com/gin-gonic/gin"
)

// TestMain configures Gin for quiet test execution before running package tests.
func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

// TestRunWithoutArgsStartsHTTPServer verifies that default execution starts the configured HTTP server.
func TestRunWithoutArgsStartsHTTPServer(t *testing.T) {
	var stdout bytes.Buffer
	var gotAddr string
	var gotRouter *gin.Engine

	app := &App{
		Stdout:     &stdout,
		Stderr:     &bytes.Buffer{},
		Environ:    []string{envHTTPAddr + "=127.0.0.1:9090"},
		DotenvPath: filepath.Join(t.TempDir(), "missing.env"),
		RunServer: func(router *gin.Engine, addr string) error {
			gotAddr = addr
			gotRouter = router
			return nil
		},
	}

	if err := app.Run(nil); err != nil {
		t.Fatalf("Run(nil) returned error: %v", err)
	}
	if gotAddr != "127.0.0.1:9090" {
		t.Fatalf("addr = %q", gotAddr)
	}
	if gotRouter == nil {
		t.Fatal("router was nil")
	}
	if !strings.Contains(stdout.String(), "listening on 127.0.0.1:9090") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

// TestRunHelpPrintsWebAPIUsage verifies that help output describes Web API startup and settings.
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

// TestRunRejectsUnknownCommand verifies that removed CLI commands are rejected.
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

// TestLoadConfigReadsDotenvAndEnvironmentOverrides verifies .env loading and process environment precedence.
func TestLoadConfigReadsDotenvAndEnvironmentOverrides(t *testing.T) {
	dotenvPath := filepath.Join(t.TempDir(), ".env")
	writeFile(t, dotenvPath, `
HTTP_ADDR=:8081
APPLE_DEVELOPER_TOKEN=token-from-file
APPLE_STOREFRONT=us
`)

	cfg, err := LoadConfig(configOptions{
		Environ: []string{
			envHTTPAddr + "=127.0.0.1:9090",
			envAppleStorefront + "=jp",
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
}

// TestLoadConfigDefaultsAppleStorefrontToJP verifies the initial storefront default.
func TestLoadConfigDefaultsAppleStorefrontToJP(t *testing.T) {
	cfg, err := LoadConfig(configOptions{DotenvPath: filepath.Join(t.TempDir(), "missing.env")})
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if cfg.AppleStorefront != "jp" {
		t.Fatalf("AppleStorefront = %q", cfg.AppleStorefront)
	}
}

// TestLoadConfigRejectsInvalidDotenv verifies configuration parse errors.
func TestLoadConfigRejectsInvalidDotenv(t *testing.T) {
	dotenvPath := filepath.Join(t.TempDir(), ".env")
	writeFile(t, dotenvPath, "BROKEN")

	_, err := LoadConfig(configOptions{DotenvPath: dotenvPath})
	if err == nil {
		t.Fatal("LoadConfig returned nil")
	}
	if !strings.Contains(err.Error(), "expected KEY=VALUE") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestHealthEndpoint verifies that the health endpoint returns a successful status payload.
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

// TestBindHandlersRegistersEndpoints verifies route binding is separate from Gin engine initialization.
func TestBindHandlersRegistersEndpoints(t *testing.T) {
	router := NewEngine()

	if got := len(router.Routes()); got != 0 {
		t.Fatalf("routes before binding = %d", got)
	}

	BindHandlers(router, Config{})
	if got := len(router.Routes()); got != 2 {
		t.Fatalf("routes after binding = %d", got)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status after binding = %d", rec.Code)
	}
}

// TestConfigEndpointRedactsDeveloperToken verifies that public config never exposes the token value.
func TestConfigEndpointRedactsDeveloperToken(t *testing.T) {
	cfg := Config{
		HTTPAddr:            ":9090",
		AppleDeveloperToken: "secret-token",
		AppleStorefront:     "jp",
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
	if got.HTTPAddr != ":9090" || got.AppleStorefront != "jp" {
		t.Fatalf("unexpected config response: %+v", got)
	}
}

// TestUnsupportedMethodsReturnMethodNotAllowed verifies method restrictions for read-only endpoints.
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

// writeFile writes test fixture content and fails the test if the write cannot be completed.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
