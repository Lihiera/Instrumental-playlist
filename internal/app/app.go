package app

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

const Name = "instrumental-playlist"

type App struct {
	Stdout     io.Writer
	Stderr     io.Writer
	Environ    []string
	DotenvPath string
	RunServer  func(router *gin.Engine, addr string) error
}

// Run creates a default application instance and starts it with the provided arguments.
func Run(args []string) error {
	return New().Run(args)
}

// New constructs an application with default OS-backed inputs and HTTP server behavior.
func New() *App {
	return &App{
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		Environ:    os.Environ(),
		DotenvPath: ".env",
		RunServer: func(router *gin.Engine, addr string) error {
			return router.Run(addr)
		},
	}
}

// Run validates command-line arguments, loads configuration, and starts the Gin server.
func (a *App) Run(args []string) error {
	if a.Stdout == nil {
		a.Stdout = io.Discard
	}
	if a.Stderr == nil {
		a.Stderr = io.Discard
	}
	if a.RunServer == nil {
		a.RunServer = func(router *gin.Engine, addr string) error {
			return router.Run(addr)
		}
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
	router := NewEngine()
	BindHandlers(router, cfg)
	return a.RunServer(router, cfg.HTTPAddr)
}

// NewEngine initializes Gin server settings without binding application handlers.
func NewEngine() *gin.Engine {
	router := gin.New()
	router.Use(gin.LoggerWithWriter(io.Discard), gin.Recovery())
	router.HandleMethodNotAllowed = true
	_ = router.SetTrustedProxies(nil)

	return router
}

// BindHandlers registers application HTTP handlers on an existing Gin engine.
func BindHandlers(router *gin.Engine, cfg Config) {
	tokens := newTokenStore()
	oauthStates := newOAuthStateStore()
	trackSearches := newTrackSearchStore()
	playlistLists := newPlaylistStore()
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/v1/config", func(c *gin.Context) {
		c.JSON(http.StatusOK, cfg.Public())
	})
	bindAuthHandlers(router, cfg, tokens, oauthStates)
	bindSpotifyHandlers(router, cfg, tokens, trackSearches, playlistLists)
}

// Handler builds a ready-to-serve Gin engine for tests and simple embedding.
func Handler(cfg Config) *gin.Engine {
	router := NewEngine()
	BindHandlers(router, cfg)
	return router
}

// writeUsage prints the command usage and supported environment variables.
func writeUsage(w io.Writer) {
	fmt.Fprintf(w, `%s

Usage:
  %s [serve]
  %s help

Environment:
  HTTP_ADDR
  SPOTIFY_CLIENT_ID
  SPOTIFY_CLIENT_SECRET
  SPOTIFY_REDIRECT_URI
  SPOTIFY_BASE_URL
  SPOTIFY_ACCOUNTS_BASE_URL

Endpoints:
  GET /health
  GET /v1/config
  POST /v1/auth/tokens
  GET /v1/auth/tokens/{tokenID}
  GET /v1/auth/status
  POST /v1/auth/logout
  GET /oauth/spotify/login
  GET /oauth/spotify/callback
  GET /v1/playlists
  POST /v1/playlists
  GET /v1/playlists/{playlistID}/tracks
  POST /v1/playlists/{playlistID}/tracks
  DELETE /v1/playlists/{playlistID}/tracks
  GET /v1/search/tracks?term=...
  GET /v1/noLogin/search/playlists?keyword=...
`, Name, Name, Name)
}
