package app

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"instrumental-playlist/internal/spotify"
)

type saveTokenRequest struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
	Scope        string `json:"scope,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
}

func bindAuthHandlers(router *gin.Engine, cfg Config, tokens *tokenStore) {
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

func expiresAt(expiresIn int) time.Time {
	if expiresIn <= 0 {
		return time.Time{}
	}
	return time.Now().UTC().Add(time.Duration(expiresIn) * time.Second)
}
