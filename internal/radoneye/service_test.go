package radoneye

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"fishnix/zero-rn/internal/cache"
	"fishnix/zero-rn/internal/config"
)

type fakeClient struct {
	results []PollResult
	errs    []error
	calls   int
}

func (f *fakeClient) ReadOnce(context.Context) (PollResult, error) {
	defer func() { f.calls++ }()
	if f.calls < len(f.errs) && f.errs[f.calls] != nil {
		return PollResult{}, f.errs[f.calls]
	}
	if f.calls < len(f.results) {
		return f.results[f.calls], nil
	}
	return PollResult{}, errors.New("no result configured")
}

func TestPollWithRetriesSuccess(t *testing.T) {
	c := cache.New()
	client := &fakeClient{
		errs: []error{errors.New("first failure"), nil},
		results: []PollResult{
			{},
			{
				RadonPcIL:  1.5,
				DeviceName: "FR:R20:SN0477",
				RSSI:       -50,
				MeasuredAt: time.Date(2026, 5, 1, 11, 0, 0, 0, time.UTC),
			},
		},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(config.Config{}, client, c, logger)

	svc.pollWithRetries(context.Background())

	state := c.Get()
	if !state.Valid {
		t.Fatalf("expected valid cache after successful retry")
	}
	if state.Reading.RadonPcIL != 1.5 {
		t.Fatalf("unexpected RadonPcIL: %v", state.Reading.RadonPcIL)
	}
	if client.calls != 2 {
		t.Fatalf("unexpected call count: got %d want 2", client.calls)
	}
}

func TestPollWithRetriesAllFail(t *testing.T) {
	c := cache.New()
	client := &fakeClient{
		errs: []error{
			errors.New("f1"),
			errors.New("f2"),
			errors.New("f3"),
			errors.New("f4"),
		},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(config.Config{}, client, c, logger)

	svc.pollWithRetries(context.Background())

	state := c.Get()
	if state.Valid {
		t.Fatalf("cache should remain invalid when all retries fail")
	}
	if state.ConsecutiveFailures != 1 {
		t.Fatalf("expected one failure mark, got %d", state.ConsecutiveFailures)
	}
	if client.calls != 4 {
		t.Fatalf("unexpected call count: got %d want 4", client.calls)
	}
}
