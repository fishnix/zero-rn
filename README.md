# zero-rn

RadonEye R200 poller for Raspberry Pi Zero W (Linux).

The service scans for an R200 over BLE, triggers a measurement, caches the latest value, and exposes it over HTTP.

## Features

- Polls RadonEye R200 over BLE (default every 30 minutes)
- Discovers devices by name prefix (`FR:R20:`) or optional fixed MAC
- Decodes the R200 payload into pCi/L and Bq/m^3
- Exposes JSON API:
  - `GET /healthz`
  - `GET /radon`
- Runs as a `systemd` service

## Requirements

- Raspberry Pi Zero W (or other Linux host with BLE)
- BlueZ (`bluetoothd`, `bluetoothctl`)
- Go toolchain (module uses `go 1.26.1`)

## Project Layout

- `cmd/radon-poller`: main application entrypoint
- `internal/radoneye`: BLE scan/connect/read logic
- `internal/httpapi`: HTTP endpoints
- `internal/cache`: in-memory reading cache
- `internal/config`: environment-driven config
- `deploy/radon-poller.service`: example `systemd` unit
- `spec/radon-poller.md`: technical specification

## Build

Local build:

```sh
go build -o bin/radon-poller ./cmd/radon-poller
```

Pi Zero W cross-build (ARMv6):

```sh
GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=0 go build -o bin/radon-poller-linux-armv6 ./cmd/radon-poller
```

## Run

```sh
RADON_HTTP_ADDR=:8080 ./bin/radon-poller
```

## Configuration

Environment variables:

- `RADON_POLL_INTERVAL` (default: `30m`)
- `RADON_SCAN_TIMEOUT` (default: `30s`)
- `RADON_READ_TIMEOUT` (default: `10s`)
- `RADON_HTTP_ADDR` (default: `:8080`)
- `RADON_NAME_PREFIX` (default: `FR:R20:`)
- `RADON_LOG_LEVEL` (default: `info`)
- `RADON_DEVICE_MAC` (optional, fixed target device)
- `RADON_HTTP_TOKEN` (optional bearer token for API auth)

## API

Health:

```sh
curl -s http://127.0.0.1:8080/healthz
```

Radon reading:

```sh
curl -s http://127.0.0.1:8080/radon
```

Example success response:

```json
{
  "status": "ok",
  "radon_pci_l": 1.63,
  "radon_bq_m3": 60.31,
  "measured_at": "2026-04-30T14:59:48Z",
  "device_name": "FR:R20:SN0477",
  "rssi_dbm": -49,
  "consecutive_failures": 0
}
```

## Deploy with systemd

Copy binary and service file:

```sh
sudo install -d -m 755 /opt/radon-poller/bin
sudo install -m 755 bin/radon-poller-linux-armv6 /opt/radon-poller/bin/radon-poller
sudo install -m 644 deploy/radon-poller.service /etc/systemd/system/radon-poller.service
```

Enable and start:

```sh
sudo systemctl daemon-reload
sudo systemctl enable --now radon-poller
```

Check status/logs:

```sh
systemctl status radon-poller
journalctl -u radon-poller -f
```

## Notes

- The service is enabled at boot when installed with `systemctl enable`.
- If `GET /radon` returns `pending`, the process is running but no successful poll has completed yet.
- BLE reliability depends on range/interference; placing the R200 close to the Pi helps during initial validation.
