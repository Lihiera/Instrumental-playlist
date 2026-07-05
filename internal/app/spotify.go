package app

import "instrumental-playlist/internal/spotify"

// SpotifyClient builds the shared Spotify Web API client from runtime configuration.
func (cfg Config) SpotifyClient() (*spotify.Client, error) {
	return spotify.New(spotify.Config{
		BaseURL: cfg.SpotifyBaseURL,
	})
}

// SpotifyAuthClient builds the Spotify Accounts API client from runtime configuration.
func (cfg Config) SpotifyAuthClient() (*spotify.AuthClient, error) {
	return spotify.NewAuthClient(spotify.AuthConfig{
		AccountsBaseURL: cfg.SpotifyAccountsBaseURL,
		ClientID:        cfg.SpotifyClientID,
		ClientSecret:    cfg.SpotifyClientSecret,
	})
}
