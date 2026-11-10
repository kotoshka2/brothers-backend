package presentation_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"brothers-backend/internal/application"
	"brothers-backend/internal/infrastructure/sqlite"
	"brothers-backend/internal/presentation"
)

func TestContentAndAuthRoutes(t *testing.T) {
	repository, err := sqlite.Open(context.Background(), filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()

	passwordHash, err := application.HashPassword("test-password-long-enough")
	if err != nil {
		t.Fatal(err)
	}
	auth := application.NewAuthService("admin", passwordHash, []byte("01234567890123456789012345678901"), repository)
	api := presentation.NewAPI(application.NewContentService(repository), auth, []string{"http://localhost:5173"})
	server := httptest.NewServer(api.Handler())
	defer server.Close()

	response := request(t, server.Client(), http.MethodGet, server.URL+"/api/v1/config", "", "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("public config status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	response.Body.Close()

	response = request(t, server.Client(), http.MethodPut, server.URL+"/api/v1/settings/time-step", `{"timeStep":15}`, "")
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized update status = %d, want %d", response.StatusCode, http.StatusUnauthorized)
	}
	response.Body.Close()

	response = request(t, server.Client(), http.MethodPost, server.URL+"/api/v1/auth/login",
		`{"username":"admin","password":"test-password-long-enough"}`, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	var tokens application.TokenPair
	if err := json.NewDecoder(response.Body).Decode(&tokens); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()

	response = request(t, server.Client(), http.MethodPut, server.URL+"/api/v1/settings/time-step",
		`{"timeStep":15}`, tokens.AccessToken)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("authenticated update status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	response.Body.Close()

	response = request(t, server.Client(), http.MethodGet, server.URL+"/api/v1/zones", "", "")
	var zones []json.RawMessage
	if response.StatusCode != http.StatusOK {
		t.Fatalf("zones status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if err := json.NewDecoder(response.Body).Decode(&zones); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if len(zones) != 7 {
		t.Fatalf("zone count = %d, want 7", len(zones))
	}

	body, err := json.Marshal(map[string]string{"refreshToken": tokens.RefreshToken})
	if err != nil {
		t.Fatal(err)
	}
	response = request(t, server.Client(), http.MethodPost, server.URL+"/api/v1/auth/logout", string(body), "")
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("logout status = %d, want %d", response.StatusCode, http.StatusNoContent)
	}
	response.Body.Close()
}

func request(t *testing.T, client *http.Client, method, url, body, accessToken string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(method, url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if accessToken != "" {
		request.Header.Set("Authorization", "Bearer "+accessToken)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}
