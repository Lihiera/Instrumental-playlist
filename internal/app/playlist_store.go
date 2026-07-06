package app

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

type playlistStore struct {
	mu     sync.RWMutex
	byUser map[string]storedPlaylistList
}

type storedPlaylistList struct {
	Items     []storedPlaylist
	CreatedAt time.Time
}

type storedPlaylist struct {
	ID   string
	Name string
	URL  string
}

func newPlaylistStore() *playlistStore {
	return &playlistStore{byUser: map[string]storedPlaylistList{}}
}

func (s *playlistStore) SaveForAccessToken(accessToken string, playlists []spotifyPlaylistSummary) {
	userKey := playlistUserKey(accessToken)
	if userKey == "" {
		return
	}

	items := make([]storedPlaylist, 0, len(playlists))
	for _, playlist := range playlists {
		items = append(items, storedPlaylist{
			ID:   strings.TrimSpace(playlist.ID),
			Name: normalizePlainTextField(playlist.Name),
			URL:  strings.TrimSpace(playlist.ExternalURLs.Spotify),
		})
	}

	s.mu.Lock()
	s.byUser[userKey] = storedPlaylistList{Items: items, CreatedAt: time.Now().UTC()}
	s.mu.Unlock()
}

func (s *playlistStore) ForAccessToken(accessToken string) (storedPlaylistList, bool) {
	userKey := playlistUserKey(accessToken)
	if userKey == "" {
		return storedPlaylistList{}, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	list, ok := s.byUser[userKey]
	if !ok {
		return storedPlaylistList{}, false
	}
	list.Items = append([]storedPlaylist(nil), list.Items...)
	return list, true
}

func (s *playlistStore) ByNumber(accessToken string, number int) (storedPlaylist, bool) {
	list, ok := s.ForAccessToken(accessToken)
	if !ok || number < 1 || number > len(list.Items) {
		return storedPlaylist{}, false
	}
	return list.Items[number-1], true
}

func playlistUserKey(accessToken string) string {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(accessToken))
	return hex.EncodeToString(sum[:])
}
