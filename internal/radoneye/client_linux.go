//go:build linux

package radoneye

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"fishnix/zero-rn/internal/config"
	"tinygo.org/x/bluetooth"
)

const (
	serviceUUIDString = "00001523-1212-efde-1523-785feabcd123"
	txUUIDString      = "00001524-1212-efde-1523-785feabcd123"
	rxUUIDString      = "00001525-1212-efde-1523-785feabcd123"
)

type LinuxClient struct {
	cfg     config.Config
	adapter *bluetooth.Adapter
	logger  *slog.Logger
	once    sync.Once
}

func NewLinuxClient(cfg config.Config, logger *slog.Logger) *LinuxClient {
	return &LinuxClient{
		cfg:     cfg,
		adapter: bluetooth.DefaultAdapter,
		logger:  logger,
	}
}

func (c *LinuxClient) ReadOnce(ctx context.Context) (PollResult, error) {
	var enableErr error
	c.once.Do(func() {
		enableErr = c.adapter.Enable()
	})
	if enableErr != nil {
		return PollResult{}, fmt.Errorf("enable bluetooth adapter: %w", enableErr)
	}

	var candidates []bluetooth.ScanResult
	var stopSoon sync.Once
	scanCtx, cancelScan := context.WithTimeout(ctx, c.cfg.ScanTimeout)
	defer cancelScan()
	go func() {
		<-scanCtx.Done()
		_ = c.adapter.StopScan()
	}()

	c.logger.Debug("starting BLE scan", "scan_timeout", c.cfg.ScanTimeout.String(), "name_prefix", c.cfg.NamePrefix, "device_mac", c.cfg.DeviceMAC)
	err := c.adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		name := strings.TrimSpace(result.LocalName())
		if c.cfg.DeviceMAC != "" && !strings.EqualFold(result.Address.String(), c.cfg.DeviceMAC) {
			return
		}
		if c.cfg.DeviceMAC == "" && !strings.HasPrefix(name, c.cfg.NamePrefix) {
			return
		}
		candidates = append(candidates, result)
		c.logger.Debug("matching scan candidate", "address", result.Address.String(), "name", name, "rssi_dbm", result.RSSI)
		stopSoon.Do(func() {
			go func() {
				// Keep scanning briefly after first hit to capture a stronger RSSI sample,
				// but avoid long scans that can destabilize the subsequent connection.
				time.Sleep(2 * time.Second)
				_ = adapter.StopScan()
			}()
		})
	})
	if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		return PollResult{}, fmt.Errorf("scan failed: %w", err)
	}

	if len(candidates) == 0 {
		return PollResult{}, fmt.Errorf("scan timeout (%s): no matching device found", c.cfg.ScanTimeout)
	}
	selected := candidates[0]
	for _, candidate := range candidates[1:] {
		if candidate.RSSI > selected.RSSI {
			selected = candidate
		}
	}
	c.logger.Info("selected scan target", "address", selected.Address.String(), "name", selected.LocalName(), "rssi_dbm", selected.RSSI, "candidate_count", len(candidates))

	device, err := c.adapter.Connect(selected.Address, bluetooth.ConnectionParams{})
	if err != nil {
		return PollResult{}, fmt.Errorf("connect to %s: %w", selected.Address.String(), err)
	}
	defer func() {
		if disconnectErr := device.Disconnect(); disconnectErr != nil {
			c.logger.Warn("device disconnect failed", "address", selected.Address.String(), "error", disconnectErr)
		}
	}()

	services, err := device.DiscoverServices(nil)
	if err != nil {
		return PollResult{}, fmt.Errorf("discover services: %w", err)
	}
	if len(services) == 0 {
		return PollResult{}, errors.New("discover services: no services found")
	}
	var radonService *bluetooth.DeviceService
	var serviceUUIDs []string
	for i := range services {
		u := strings.ToLower(services[i].UUID().String())
		serviceUUIDs = append(serviceUUIDs, u)
		if u == serviceUUIDString {
			radonService = &services[i]
			break
		}
	}
	if radonService == nil {
		return PollResult{}, fmt.Errorf("discover service: radoneye service not found (services=%v)", serviceUUIDs)
	}

	chars, err := radonService.DiscoverCharacteristics(nil)
	if err != nil {
		return PollResult{}, fmt.Errorf("discover characteristics: %w", err)
	}

	var tx, rx *bluetooth.DeviceCharacteristic
	for i := range chars {
		ch := &chars[i]
		u := strings.ToLower(ch.UUID().String())
		switch u {
		case txUUIDString:
			tx = ch
		case rxUUIDString:
			rx = ch
		}
	}
	if tx == nil || rx == nil {
		return PollResult{}, errors.New("missing tx or rx characteristic")
	}

	resultCh := make(chan PollResult, 1)
	errCh := make(chan error, 1)
	if err := rx.EnableNotifications(func(payload []byte) {
		pciL, err := decodeRadonPcIL(payload)
		if err != nil {
			select {
			case errCh <- err:
			default:
			}
			return
		}
		select {
		case resultCh <- PollResult{
			RadonPcIL:  pciL,
			DeviceName: strings.TrimSpace(selected.LocalName()),
			RSSI:       selected.RSSI,
			MeasuredAt: time.Now(),
		}:
		default:
		}
	}); err != nil {
		return PollResult{}, fmt.Errorf("enable notifications: %w", err)
	}

	if _, err := tx.WriteWithoutResponse([]byte{0x50}); err != nil {
		return PollResult{}, fmt.Errorf("write trigger: %w", err)
	}

	readCtx, cancelRead := context.WithTimeout(ctx, c.cfg.ReadTimeout)
	defer cancelRead()

	select {
	case <-readCtx.Done():
		return PollResult{}, fmt.Errorf("read timeout (%s): %w", c.cfg.ReadTimeout, readCtx.Err())
	case err := <-errCh:
		return PollResult{}, fmt.Errorf("decode notification: %w", err)
	case result := <-resultCh:
		return result, nil
	}
}
