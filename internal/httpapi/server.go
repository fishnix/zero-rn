package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"fishnix/zero-rn/internal/cache"
	"fishnix/zero-rn/internal/config"
)

type Server struct {
	cfg    config.Config
	cache  *cache.Cache
	logger *slog.Logger
	mux    *http.ServeMux
}

func NewServer(cfg config.Config, c *cache.Cache, logger *slog.Logger) *Server {
	s := &Server{
		cfg:    cfg,
		cache:  c,
		logger: logger,
		mux:    http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.cfg.HTTPToken != "" && !isAuthorized(r, s.cfg.HTTPToken) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"status": "unauthorized",
		})
		return
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("/healthz", s.handleHealth)
	s.mux.HandleFunc("/radon", s.handleRadon)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleRadon(w http.ResponseWriter, _ *http.Request) {
	state := s.cache.Get()
	if !state.Valid {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status":               "pending",
			"radon_pci_l":          nil,
			"radon_bq_m3":          nil,
			"measured_at":          nil,
			"consecutive_failures": state.ConsecutiveFailures,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":               "ok",
		"radon_pci_l":          state.Reading.RadonPcIL,
		"radon_bq_m3":          state.Reading.RadonBqM3,
		"measured_at":          state.Reading.MeasuredAt.UTC().Format(time.RFC3339),
		"device_name":          state.Reading.DeviceName,
		"rssi_dbm":             state.Reading.RSSI,
		"consecutive_failures": state.ConsecutiveFailures,
	})
}

func isAuthorized(r *http.Request, expected string) bool {
	const bearer = "Bearer "
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(authHeader, bearer) {
		return false
	}
	token := strings.TrimPrefix(authHeader, bearer)
	return token == expected
}

func writeJSON(w http.ResponseWriter, status int, payload map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
