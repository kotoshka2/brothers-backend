package presentation

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"brothers-backend/internal/application"
	"brothers-backend/internal/domain"
)

type API struct {
	content      *application.ContentService
	auth         *application.AuthService
	allowedHosts map[string]struct{}
}

func NewAPI(content *application.ContentService, auth *application.AuthService, allowedOrigins []string) *API {
	origins := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			origins[origin] = struct{}{}
		}
	}
	return &API{content: content, auth: auth, allowedHosts: origins}
}

func (api *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", api.health)
	mux.HandleFunc("POST /api/v1/auth/login", api.login)
	mux.HandleFunc("POST /api/v1/auth/refresh", api.refresh)
	mux.HandleFunc("POST /api/v1/auth/logout", api.logout)

	mux.HandleFunc("GET /api/v1/config", api.getConfig)
	mux.Handle("PUT /api/v1/config", api.requireAuth(http.HandlerFunc(api.updateConfig)))
	mux.HandleFunc("GET /api/v1/settings/time-step", api.getSettings)
	mux.Handle("PUT /api/v1/settings/time-step", api.requireAuth(http.HandlerFunc(api.updateSettings)))

	mux.HandleFunc("GET /api/v1/zones", api.listZones)
	mux.Handle("POST /api/v1/zones", api.requireAuth(http.HandlerFunc(api.createZone)))
	mux.HandleFunc("GET /api/v1/zones/{id}", api.getZone)
	mux.Handle("PUT /api/v1/zones/{id}", api.requireAuth(http.HandlerFunc(api.updateZone)))
	mux.Handle("DELETE /api/v1/zones/{id}", api.requireAuth(http.HandlerFunc(api.deleteZone)))

	return api.cors(mux)
}

func (api *API) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (api *API) login(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	tokens, err := api.auth.Login(r.Context(), request.Username, request.Password)
	if err != nil {
		api.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tokens)
}

func (api *API) refresh(w http.ResponseWriter, r *http.Request) {
	var request struct {
		RefreshToken string `json:"refreshToken"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	tokens, err := api.auth.Refresh(r.Context(), request.RefreshToken)
	if err != nil {
		api.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tokens)
}

func (api *API) logout(w http.ResponseWriter, r *http.Request) {
	var request struct {
		RefreshToken string `json:"refreshToken"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if err := api.auth.Logout(r.Context(), request.RefreshToken); err != nil {
		api.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *API) getConfig(w http.ResponseWriter, r *http.Request) {
	config, err := api.content.GetConfig(r.Context())
	if err != nil {
		api.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, config)
}

func (api *API) updateConfig(w http.ResponseWriter, r *http.Request) {
	var config domain.Config
	if !decodeJSON(w, r, &config) {
		return
	}
	if err := api.content.UpdateConfig(r.Context(), config); err != nil {
		api.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, config)
}

func (api *API) getSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := api.content.GetSettings(r.Context())
	if err != nil {
		api.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (api *API) updateSettings(w http.ResponseWriter, r *http.Request) {
	var settings domain.Settings
	if !decodeJSON(w, r, &settings) {
		return
	}
	if err := api.content.UpdateSettings(r.Context(), settings); err != nil {
		api.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (api *API) listZones(w http.ResponseWriter, r *http.Request) {
	zones, err := api.content.ListZones(r.Context())
	if err != nil {
		api.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, zones)
}

func (api *API) getZone(w http.ResponseWriter, r *http.Request) {
	zone, err := api.content.GetZone(r.Context(), r.PathValue("id"))
	if err != nil {
		api.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, zone)
}

func (api *API) createZone(w http.ResponseWriter, r *http.Request) {
	var zone domain.Zone
	if !decodeJSON(w, r, &zone) {
		return
	}
	if err := api.content.CreateZone(r.Context(), zone); err != nil {
		api.writeError(w, err)
		return
	}
	w.Header().Set("Location", "/api/v1/zones/"+zone.ID)
	writeJSON(w, http.StatusCreated, zone)
}

func (api *API) updateZone(w http.ResponseWriter, r *http.Request) {
	var zone domain.Zone
	if !decodeJSON(w, r, &zone) {
		return
	}
	if zone.ID != r.PathValue("id") {
		api.writeError(w, application.ErrInvalid)
		return
	}
	if err := api.content.UpdateZone(r.Context(), zone); err != nil {
		api.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, zone)
}

func (api *API) deleteZone(w http.ResponseWriter, r *http.Request) {
	if err := api.content.DeleteZone(r.Context(), r.PathValue("id")); err != nil {
		api.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *API) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Fields(r.Header.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") ||
			api.auth.ValidateAccessToken(parts[1]) != nil {
			api.writeError(w, application.ErrUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (api *API) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if _, allowed := api.allowedHosts[origin]; allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				w.Header().Set("Vary", "Origin")
			}
		}
		if r.Method == http.MethodOptions {
			if origin != "" {
				if _, allowed := api.allowedHosts[origin]; !allowed {
					http.Error(w, "origin is not allowed", http.StatusForbidden)
					return
				}
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (api *API) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, application.ErrInvalid):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	case errors.Is(err, application.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "resource not found"})
	case errors.Is(err, application.ErrConflict):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "resource already exists"})
	case errors.Is(err, application.ErrUnauthorized):
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	default:
		log.Printf("api request failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request body must contain one JSON value"})
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write response failed: %v", err)
	}
}
