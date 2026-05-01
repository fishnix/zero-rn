package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fishnix/zero-rn/internal/cache"
	"fishnix/zero-rn/internal/config"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHealthz(t *testing.T) {
	s := NewServer(config.Config{}, cache.New(), testLogger())

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d want %d", rr.Code, http.StatusOK)
	}
}

func TestRadonPending(t *testing.T) {
	s := NewServer(config.Config{}, cache.New(), testLogger())

	req := httptest.NewRequest(http.MethodGet, "/radon", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status mismatch: got %d want %d", rr.Code, http.StatusServiceUnavailable)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if got, want := body["status"], "pending"; got != want {
		t.Fatalf("status field mismatch: got %v want %v", got, want)
	}
}

func TestRadonOk(t *testing.T) {
	c := cache.New()
	c.SetReading(cache.Reading{
		RadonPcIL:  1.63,
		RadonBqM3:  60.31,
		DeviceName: "FR:R20:SN0477",
		RSSI:       -49,
		MeasuredAt: time.Date(2026, 4, 30, 14, 59, 48, 0, time.UTC),
	})
	s := NewServer(config.Config{}, c, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/radon", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d want %d", rr.Code, http.StatusOK)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if got, want := body["status"], "ok"; got != want {
		t.Fatalf("status field mismatch: got %v want %v", got, want)
	}
	if got := body["device_name"]; got == nil {
		t.Fatalf("expected device_name in response")
	}
}

func TestTokenAuth(t *testing.T) {
	cfg := config.Config{HTTPToken: "secret"}
	s := NewServer(cfg, cache.New(), testLogger())

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized, got %d", rr.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req2.Header.Set("Authorization", "Bearer secret")
	rr2 := httptest.NewRecorder()
	s.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected OK with token, got %d", rr2.Code)
	}
}
