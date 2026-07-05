package app

import "instrumental-playlist/internal/spotify"

// SpotifyClient builds the shared Spotify Web API client from runtime configuration.
func (cfg Config) SpotifyClient() (*spotify.Client, error) {
	return spotify.New(spotify.Config{
		BaseURL: cfg.SpotifyBaseURL,
	})
}
