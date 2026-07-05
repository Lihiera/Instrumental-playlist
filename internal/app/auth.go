package app

import (
	"bytes"
	"errors"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"instrumental-playlist/internal/spotify"
)

var spotifyOAuthScopes = []string{
	"playlist-read-private",
	"playlist-modify-public",
	"playlist-modify-private",
}

var oauthSuccessPageTemplate = template.Must(template.New("oauth-success").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Spotify Login Complete</title>
  <style>
    :root { color-scheme: light dark; font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
    body { margin: 0; min-height: 100vh; display: grid; place-items: center; background: #f6f8fa; color: #1f2328; }
    main { width: min(560px, calc(100vw - 32px)); padding: 32px; border: 1px solid #d0d7de; border-radius: 8px; background: #ffffff; box-shadow: 0 12px 32px rgba(31, 35, 40, 0.08); }
    h1 { margin: 0 0 12px; font-size: 28px; line-height: 1.2; }
    p { margin: 0 0 20px; line-height: 1.6; }
    dl { display: grid; grid-template-columns: max-content 1fr; gap: 10px 16px; margin: 0; }
    dt { font-weight: 700; color: #57606a; }
    dd { margin: 0; overflow-wrap: anywhere; }
    code { font-family: ui-monospace, SFMono-Regular, Consolas, "Liberation Mono", monospace; }
    a { color: #0969da; }
    @media (prefers-color-scheme: dark) {
      body { background: #0d1117; color: #e6edf3; }
      main { background: #161b22; border-color: #30363d; box-shadow: none; }
      dt { color: #8b949e; }
      a { color: #58a6ff; }
    }
  </style>
</head>
<body>
  <main>
    <h1>Spotify login complete</h1>
    <p>User token metadata was saved in process memory. Access and refresh token values are hidden.</p>
    <dl>
      <dt>Token ID</dt>
      <dd><code>{{.ID}}</code></dd>
      <dt>Token type</dt>
      <dd>{{.TokenType}}</dd>
      <dt>Scope</dt>
      <dd>{{.Scope}}</dd>
      <dt>Expires at</dt>
      <dd>{{.ExpiresAt}}</dd>
      <dt>Refresh token</dt>
      <dd>{{if .HasRefreshToken}}Saved{{else}}Not provided{{end}}</dd>
    </dl>
    <p style="margin-top: 24px;">Metadata API: <a href="/v1/auth/tokens/{{.ID}}">/v1/auth/tokens/{{.ID}}</a></p>
  </main>
</body>
</html>`))

type saveTokenRequest struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
	Scope        string `json:"scope,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
}

type authStatusResponse struct {
	LoggedIn           bool           `json:"logged_in"`
	Token              *tokenMetadata `json:"token,omitempty"`
	AccessTokenExpired bool           `json:"access_token_expired,omitempty"`
}

type logoutResponse struct {
	LoggedOut bool `json:"logged_out"`
}

func bindAuthHandlers(router *gin.Engine, cfg Config, tokens *tokenStore, oauthStates *oauthStateStore) {
	router.POST("/v1/auth/tokens", func(c *gin.Context) {
		var req saveTokenRequest
		if !bindJSON(c, &req) {
			return
		}
		accessToken := strings.TrimSpace(req.AccessToken)
		if accessToken == "" {
			writeAPIError(c, http.StatusBadRequest, "invalid_request", "access_token is required", 0)
			return
		}

		metadata, err := tokens.Save(storedToken{
			AccessToken:  accessToken,
			RefreshToken: strings.TrimSpace(req.RefreshToken),
			TokenType:    strings.TrimSpace(req.TokenType),
			Scope:        strings.TrimSpace(req.Scope),
			ExpiresAt:    expiresAt(req.ExpiresIn),
		})
		if err != nil {
			writeAPIError(c, http.StatusInternalServerError, "token_store_failed", "token could not be saved in memory", 0)
			return
		}
		c.JSON(http.StatusCreated, metadata)
	})

	router.GET("/v1/auth/tokens/:tokenID", func(c *gin.Context) {
		token, ok := tokens.Get(c.Param("tokenID"))
		if !ok {
			writeAPIError(c, http.StatusNotFound, "token_not_found", "token was not found in memory", 0)
			return
		}
		c.JSON(http.StatusOK, metadataFor(token))
	})

	router.GET("/v1/auth/status", func(c *gin.Context) {
		token, ok := tokens.Latest()
		if !ok {
			c.JSON(http.StatusOK, authStatusResponse{LoggedIn: false})
			return
		}
		metadata := metadataFor(token)
		c.JSON(http.StatusOK, authStatusResponse{
			LoggedIn:           true,
			Token:              &metadata,
			AccessTokenExpired: accessTokenExpired(token, time.Now().UTC()),
		})
	})

	router.POST("/v1/auth/logout", func(c *gin.Context) {
		c.JSON(http.StatusOK, logoutResponse{LoggedOut: tokens.Clear()})
	})

	router.GET("/oauth/spotify/login", func(c *gin.Context) {
		client, ok := spotifyAuthRequest(c, cfg)
		if !ok {
			return
		}
		if strings.TrimSpace(cfg.SpotifyRedirectURI) == "" {
			writeAPIError(c, http.StatusServiceUnavailable, "spotify_redirect_uri_missing", "spotify redirect uri is not configured", 0)
			return
		}

		state, err := oauthStates.Create()
		if err != nil {
			writeAPIError(c, http.StatusInternalServerError, "spotify_oauth_state_failed", "oauth state could not be saved in memory", 0)
			return
		}

		c.Redirect(http.StatusFound, client.AuthorizationURL(cfg.SpotifyRedirectURI, state, spotifyOAuthScopes))
	})

	router.GET("/oauth/spotify/callback", func(c *gin.Context) {
		if !oauthStates.Consume(strings.TrimSpace(c.Query("state"))) {
			writeAPIError(c, http.StatusBadRequest, "spotify_oauth_state_invalid", "OAuth callback state is missing, expired, or invalid", 0)
			return
		}

		code := strings.TrimSpace(c.Query("code"))
		if code == "" {
			writeAPIError(c, http.StatusBadRequest, "spotify_oauth_code_missing", "OAuth callback code is missing", 0)
			return
		}
		if strings.TrimSpace(cfg.SpotifyRedirectURI) == "" {
			writeAPIError(c, http.StatusServiceUnavailable, "spotify_redirect_uri_missing", "spotify redirect uri is not configured", 0)
			return
		}

		client, ok := spotifyAuthRequest(c, cfg)
		if !ok {
			return
		}
		token, err := client.AuthorizationCodeToken(c.Request.Context(), code, cfg.SpotifyRedirectURI)
		if err != nil {
			writeAuthError(c, err)
			return
		}
		if strings.TrimSpace(token.AccessToken) == "" {
			writeAPIError(c, http.StatusBadGateway, "spotify_auth_request_failed", "spotify auth response did not include an access token", 0)
			return
		}

		metadata, err := tokens.Save(storedToken{
			AccessToken:  strings.TrimSpace(token.AccessToken),
			RefreshToken: strings.TrimSpace(token.RefreshToken),
			TokenType:    strings.TrimSpace(token.TokenType),
			Scope:        strings.TrimSpace(token.Scope),
			ExpiresAt:    expiresAt(token.ExpiresIn),
		})
		if err != nil {
			writeAPIError(c, http.StatusInternalServerError, "token_store_failed", "token could not be saved in memory", 0)
			return
		}
		writeOAuthSuccessPage(c, metadata)
	})
}

func writeAuthConfigError(c *gin.Context, err error) {
	if errors.Is(err, spotify.ErrMissingClientCredentials) {
		writeAPIError(c, http.StatusServiceUnavailable, "spotify_client_credentials_missing", "spotify client credentials are not configured", 0)
		return
	}
	writeAPIError(c, http.StatusInternalServerError, "spotify_auth_config_invalid", "spotify auth configuration is invalid", 0)
}

func writeAuthError(c *gin.Context, err error) {
	var authErr *spotify.AuthError
	if errors.As(err, &authErr) {
		status := authErr.StatusCode
		if status < http.StatusBadRequest || status >= 600 {
			status = http.StatusBadGateway
		}
		message := authErr.Description
		if message == "" {
			message = "spotify client credentials request failed"
		}
		writeAPIError(c, status, "spotify_auth_error", message, authErr.StatusCode)
		return
	}
	writeAPIError(c, http.StatusBadGateway, "spotify_auth_request_failed", "spotify auth request failed", 0)
}

func spotifyAuthRequest(c *gin.Context, cfg Config) (*spotify.AuthClient, bool) {
	client, err := cfg.SpotifyAuthClient()
	if err != nil {
		writeAuthConfigError(c, err)
		return nil, false
	}
	return client, true
}

func writeOAuthSuccessPage(c *gin.Context, metadata tokenMetadata) {
	var body bytes.Buffer
	if err := oauthSuccessPageTemplate.Execute(&body, metadata); err != nil {
		writeAPIError(c, http.StatusInternalServerError, "oauth_success_page_failed", "oauth success page could not be rendered", 0)
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", body.Bytes())
}

func expiresAt(expiresIn int) time.Time {
	if expiresIn <= 0 {
		return time.Time{}
	}
	return time.Now().UTC().Add(time.Duration(expiresIn) * time.Second)
}

func accessTokenExpired(token storedToken, now time.Time) bool {
	return !token.ExpiresAt.IsZero() && !token.ExpiresAt.After(now)
}
