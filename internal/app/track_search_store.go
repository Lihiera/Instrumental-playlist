package app

import (
	"strings"
	"sync"
	"time"
)

type trackSearchStore struct {
	mu     sync.RWMutex
	latest storedTrackSearch
}

type storedTrackSearch struct {
	Term      string
	Items     []trackSearchItem
	CreatedAt time.Time
}

func newTrackSearchStore() *trackSearchStore {
	return &trackSearchStore{}
}

func (s *trackSearchStore) Save(term string, items []trackSearchItem) storedTrackSearch {
	copied := append([]trackSearchItem(nil), items...)
	search := storedTrackSearch{
		Term:      strings.TrimSpace(term),
		Items:     copied,
		CreatedAt: time.Now().UTC(),
	}

	s.mu.Lock()
	s.latest = search
	s.mu.Unlock()

	return search
}

func (s *trackSearchStore) Latest() (storedTrackSearch, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.latest.CreatedAt.IsZero() {
		return storedTrackSearch{}, false
	}
	search := s.latest
	search.Items = append([]trackSearchItem(nil), s.latest.Items...)
	return search, true
}
