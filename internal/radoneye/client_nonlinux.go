//go:build !linux

package radoneye

import (
	"context"
	"errors"
	"log/slog"

	"fishnix/zero-rn/internal/config"
)

type LinuxClient struct{}

func NewLinuxClient(_ config.Config, _ *slog.Logger) *LinuxClient {
	return &LinuxClient{}
}

func (c *LinuxClient) ReadOnce(_ context.Context) (PollResult, error) {
	return PollResult{}, errors.New("ble polling is only supported on linux builds")
}
