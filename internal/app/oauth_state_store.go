package app

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

const oauthStateTTL = 10 * time.Minute

type oauthStateStore struct {
	mu     sync.Mutex
	states map[string]time.Time
	now    func() time.Time
}

func newOAuthStateStore() *oauthStateStore {
	return &oauthStateStore{
		states: map[string]time.Time{},
		now:    func() time.Time { return time.Now().UTC() },
	}
}

func (s *oauthStateStore) Create() (string, error) {
	state, err := randomURLSafeSecret()
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	s.states[state] = s.now().Add(oauthStateTTL)
	s.mu.Unlock()

	return state, nil
}

func (s *oauthStateStore) Consume(state string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	expiresAt, ok := s.states[state]
	if !ok {
		return false
	}
	delete(s.states, state)
	return s.now().Before(expiresAt)
}

func randomURLSafeSecret() (string, error) {
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate oauth state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}
