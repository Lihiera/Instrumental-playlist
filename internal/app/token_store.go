package app

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

type tokenStore struct {
	mu            sync.RWMutex
	tokens        map[string]storedToken
	latestTokenID string
}

type storedToken struct {
	ID           string
	AccessToken  string
	RefreshToken string
	TokenType    string
	Scope        string
	ExpiresAt    time.Time
	CreatedAt    time.Time
}

type tokenMetadata struct {
	ID              string    `json:"id"`
	TokenType       string    `json:"token_type,omitempty"`
	Scope           string    `json:"scope,omitempty"`
	ExpiresAt       time.Time `json:"expires_at,omitempty"`
	HasRefreshToken bool      `json:"has_refresh_token"`
}

func newTokenStore() *tokenStore {
	return &tokenStore{tokens: map[string]storedToken{}}
}

func (s *tokenStore) Save(token storedToken) (tokenMetadata, error) {
	id, err := randomTokenID()
	if err != nil {
		return tokenMetadata{}, err
	}
	now := time.Now().UTC()
	token.ID = id
	token.CreatedAt = now

	s.mu.Lock()
	s.tokens[id] = token
	s.latestTokenID = id
	s.mu.Unlock()

	return metadataFor(token), nil
}

func (s *tokenStore) Get(id string) (storedToken, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	token, ok := s.tokens[id]
	return token, ok
}

func (s *tokenStore) Latest() (storedToken, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	token, ok := s.tokens[s.latestTokenID]
	return token, ok
}

func (s *tokenStore) Clear() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	hadTokens := len(s.tokens) > 0
	s.tokens = map[string]storedToken{}
	s.latestTokenID = ""
	return hadTokens
}

func metadataFor(token storedToken) tokenMetadata {
	return tokenMetadata{
		ID:              token.ID,
		TokenType:       token.TokenType,
		Scope:           token.Scope,
		ExpiresAt:       token.ExpiresAt,
		HasRefreshToken: token.RefreshToken != "",
	}
}

func randomTokenID() (string, error) {
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate token id: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}
