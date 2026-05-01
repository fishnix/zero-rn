package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fishnix/zero-rn/internal/cache"
	"fishnix/zero-rn/internal/config"
	"fishnix/zero-rn/internal/httpapi"
	"fishnix/zero-rn/internal/radoneye"
)

func main() {
	cfg := config.Load()
	logger := config.NewLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	c := cache.New()
	client := radoneye.NewLinuxClient(cfg, logger)
	service := radoneye.NewService(cfg, client, c, logger)

	go service.Run(ctx)

	server := httpapi.NewServer(cfg, c, logger)
	httpSrv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           server,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := httpSrv.Shutdown(shutdownCtx); err != nil {
			logger.Error("http shutdown failed", "error", err)
		}
	}()

	logger.Info("starting radon poller", "http_addr", cfg.HTTPAddr, "poll_interval", cfg.PollInterval.String())
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("http server failed", "error", err)
		os.Exit(1)
	}
}
