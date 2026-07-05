package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSaveUserTokenStoresMetadataWithoutReturningRefreshToken(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/tokens", strings.NewReader(`{
		"access_token":"user-access-token",
		"refresh_token":"user-refresh-token",
		"token_type":"bearer",
		"scope":"playlist-read-private",
		"expires_in":3600
	}`))
	req.Header.Set("Content-Type", "application/json")

	Handler(Config{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "user-access-token") || strings.Contains(rec.Body.String(), "user-refresh-token") {
		t.Fatalf("response leaked token: %s", rec.Body.String())
	}

	var got tokenMetadata
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID == "" || !got.HasRefreshToken || got.Scope != "playlist-read-private" {
		t.Fatalf("unexpected token metadata: %+v", got)
	}
}

func TestSavedTokenMetadataCanBeReadFromMemory(t *testing.T) {
	router := Handler(Config{})
	save := httptest.NewRecorder()
	saveReq := httptest.NewRequest(http.MethodPost, "/v1/auth/tokens", strings.NewReader(`{"access_token":"user-access-token"}`))
	saveReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(save, saveReq)
	if save.Code != http.StatusCreated {
		t.Fatalf("save status = %d body = %s", save.Code, save.Body.String())
	}

	var saved tokenMetadata
	if err := json.Unmarshal(save.Body.Bytes(), &saved); err != nil {
		t.Fatalf("decode save response: %v", err)
	}

	get := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/v1/auth/tokens/"+saved.ID, nil)
	router.ServeHTTP(get, getReq)
	if get.Code != http.StatusOK {
		t.Fatalf("get status = %d body = %s", get.Code, get.Body.String())
	}
	if strings.Contains(get.Body.String(), "user-access-token") {
		t.Fatalf("metadata response leaked token: %s", get.Body.String())
	}
}
