package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const Name = "instrumental-playlist"

type App struct {
	Stdout         io.Writer
	Stderr         io.Writer
	Environ        []string
	DotenvPath     string
	ListenAndServe func(addr string, handler http.Handler) error
}

func Run(args []string) error {
	return New().Run(args)
}

func New() *App {
	return &App{
		Stdout:         os.Stdout,
		Stderr:         os.Stderr,
		Environ:        os.Environ(),
		DotenvPath:     ".env",
		ListenAndServe: http.ListenAndServe,
	}
}

func (a *App) Run(args []string) error {
	if a.Stdout == nil {
		a.Stdout = io.Discard
	}
	if a.Stderr == nil {
		a.Stderr = io.Discard
	}
	if a.ListenAndServe == nil {
		a.ListenAndServe = http.ListenAndServe
	}

	if len(args) > 0 {
		switch args[0] {
		case "help", "-h", "--help":
			writeUsage(a.Stdout)
			return nil
		case "serve":
			if len(args) > 1 {
				return fmt.Errorf("%s: serve does not accept arguments", Name)
			}
		default:
			return fmt.Errorf("%s: unknown command %q", Name, args[0])
		}
	}

	cfg, err := LoadConfig(configOptions{
		Environ:    a.Environ,
		DotenvPath: a.DotenvPath,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(a.Stdout, "%s listening on %s\n", Name, cfg.HTTPAddr)
	return a.ListenAndServe(cfg.HTTPAddr, Handler(cfg))
}

func Handler(cfg Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/v1/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, cfg.Public())
	})
	return mux
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeUsage(w io.Writer) {
	fmt.Fprintf(w, `%s

Usage:
  %s [serve]
  %s help

Environment:
  HTTP_ADDR
  APPLE_DEVELOPER_TOKEN
  APPLE_STOREFRONT
  INSTRUMENTAL_THRESHOLD

Endpoints:
  GET /health
  GET /v1/config
`, Name, Name, Name)
}
