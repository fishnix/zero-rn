package radoneye

import (
	"context"
	"log/slog"
	"time"

	"fishnix/zero-rn/internal/cache"
	"fishnix/zero-rn/internal/config"
)

type PollResult struct {
	RadonPcIL  float32
	DeviceName string
	RSSI       int16
	MeasuredAt time.Time
}

type Client interface {
	ReadOnce(ctx context.Context) (PollResult, error)
}

type Service struct {
	cfg    config.Config
	client Client
	cache  *cache.Cache
	logger *slog.Logger
}

func NewService(cfg config.Config, client Client, c *cache.Cache, logger *slog.Logger) *Service {
	return &Service{
		cfg:    cfg,
		client: client,
		cache:  c,
		logger: logger,
	}
}

func (s *Service) Run(ctx context.Context) {
	s.pollWithRetries(ctx)

	ticker := time.NewTicker(s.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("poll loop stopped")
			return
		case <-ticker.C:
			s.pollWithRetries(ctx)
		}
	}
}

func (s *Service) pollWithRetries(ctx context.Context) {
	backoffs := []time.Duration{0, time.Second, 2 * time.Second, 4 * time.Second}
	var lastErr error

	for attempt, backoff := range backoffs {
		if backoff > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
		}

		if attempt > 0 {
			s.logger.Warn("retrying poll cycle", "attempt", attempt+1)
		}

		result, err := s.client.ReadOnce(ctx)
		if err != nil {
			lastErr = err
			s.logger.Warn("poll cycle failed", "attempt", attempt+1, "error", err)
			continue
		}

		s.cache.SetReading(cache.Reading{
			RadonPcIL:  result.RadonPcIL,
			RadonBqM3:  result.RadonPcIL * 37.0,
			DeviceName: result.DeviceName,
			RSSI:       result.RSSI,
			MeasuredAt: result.MeasuredAt.UTC(),
		})
		s.logger.Info("poll cycle succeeded", "radon_pci_l", result.RadonPcIL, "device_name", result.DeviceName, "rssi_dbm", result.RSSI)
		return
	}

	s.cache.MarkFailure()
	if lastErr != nil {
		s.logger.Error("all poll attempts failed", "error", lastErr)
	}
}
