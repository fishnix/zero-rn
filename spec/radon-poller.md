# Radon Poller — Technical Specification (Pi Zero W / Linux)

Service running on a Raspberry Pi Zero W (Linux) that polls a RadonEye R200 over Bluetooth Low Energy (BLE), caches the latest reading, and serves results over HTTP on the local network.

---

## 1. Overview

The Radon Poller runs as a long-lived Linux process and has two responsibilities:

1. **BLE poller**: at a fixed interval (default: 30 minutes), discover/connect to an R200, request a reading, decode it, and refresh cache state.
2. **HTTP API**: continuously expose the latest reading via `GET /radon`.

Primary implementation language is **Go**. TinyGo is acceptable if Linux BLE behavior and dependencies remain compatible.

---

## 2. Deployment Target

| Component | Detail |
|---|---|
| Host | Raspberry Pi Zero W |
| OS | Raspberry Pi OS Lite (or equivalent Debian-based Linux) |
| Runtime | `systemd` service |
| Bluetooth stack | BlueZ (via D-Bus) |
| Network | WiFi (wlan0) |
| Sensor | RadonEye R200 |

### Out of scope

- Any firmware flashing, UF2 workflow, or microcontroller-specific setup.
- Any Raspberry Pi Pico W / Pico 2W references or constraints.

---

## 3. RadonEye R200 BLE Protocol

Behavior and UUID hints are based on reverse engineering in [fishnix/radoneyebt](https://github.com/fishnix/radoneyebt).

### UUIDs

| Role | UUID |
|---|---|
| Service | `00001523-1212-efde-1523-785feabcd123` |
| TX (write trigger) | `00001524-1212-efde-1523-785feabcd123` |
| RX (notify/read result) | `00001525-1212-efde-1523-785feabcd123` |

### Discovery strategy

- Prefer matching advertisements by local name prefix: `FR:R20:` (example: `FR:R20:SN0477`).
- Do not rely exclusively on service UUID being present in advertisement payload.
- After connection, verify the expected service UUID before issuing the trigger write.

### Reading sequence

1. Scan for candidate devices for up to `scanTimeout` (default: 30s).
2. Select first matching advertisement (`FR:R20:` prefix, strongest RSSI if multiple).
3. Connect.
4. Discover service and TX/RX characteristics.
5. Subscribe to RX notifications.
6. Write single byte `0x50` to TX to trigger measurement.
7. Wait up to `readTimeout` (default: 10s) for a notification payload.
8. Decode bytes `[2:6]` as little-endian `float32` in **pCi/L**.
9. Disconnect.

### Unit conversion

`Bq/m^3 = pCi/L * 37.0`

---

## 4. Software Architecture

### Execution model

- One process with:
  - `poller` goroutine (ticker-driven BLE cycle)
  - `http` goroutine (`net/http` server)
  - shared in-memory cache protected by mutex/RWMutex

### Suggested package layout

```text
zero-rn/
├── cmd/radon-poller/
│   └── main.go
├── internal/radoneye/
│   ├── client.go        # scan/connect/read workflow
│   └── decode.go        # payload decode + unit conversion
├── internal/cache/
│   └── cache.go         # thread-safe latest reading state
├── internal/httpapi/
│   └── server.go        # /healthz and /radon handlers
├── internal/config/
│   └── config.go        # env + defaults
└── spec/
    └── radon-poller.md
```

### Cache model

```go
type Reading struct {
    RadonPcIL float32
    RadonBqM3 float32
    DeviceName string
    RSSI int16
    MeasuredAt time.Time
}

type Cache struct {
    mu sync.RWMutex
    reading Reading
    valid bool
    consecutiveFailures int
}
```

---

## 5. HTTP Interface

### Endpoints

- `GET /healthz`: liveness check, always 200 once server starts.
- `GET /radon`: latest sensor reading and metadata.

### `GET /radon` (reading available)

Status: `200 OK`

```json
{
  "status": "ok",
  "radon_pci_l": 1.27,
  "radon_bq_m3": 46.99,
  "measured_at": "2026-04-30T09:10:00Z",
  "device_name": "FR:R20:SN0477",
  "rssi_dbm": -67,
  "consecutive_failures": 0
}
```

### `GET /radon` (no successful reading yet)

Status: `503 Service Unavailable`

```json
{
  "status": "pending",
  "radon_pci_l": null,
  "radon_bq_m3": null,
  "measured_at": null,
  "consecutive_failures": 2
}
```

### Security note

No auth in v1. Bind to LAN only (`0.0.0.0:8080` or restricted interface) and place behind trusted network boundaries.

---

## 6. Configuration

Configuration via environment variables (with defaults):

- `RADON_POLL_INTERVAL=30m`
- `RADON_SCAN_TIMEOUT=30s`
- `RADON_READ_TIMEOUT=10s`
- `RADON_HTTP_ADDR=:8080`
- `RADON_NAME_PREFIX=FR:R20:`
- `RADON_LOG_LEVEL=info`

Optional:

- `RADON_DEVICE_MAC=` (if set, skip discovery and connect directly)
- `RADON_HTTP_TOKEN=` (if non-empty, require bearer token in API calls)

---

## 7. Implementation Notes (Go and TinyGo)

### Preferred: Go on Linux

- Use a Linux BLE library that supports BlueZ/D-Bus central mode.
- Use standard library `net/http`.
- Run as a `systemd` unit with restart policy.

### Optional: TinyGo on Linux

- TinyGo may be used if selected BLE and HTTP dependencies compile and behave correctly on Linux.
- If TinyGo introduces runtime/library limitations on Linux BLE, fall back to standard Go for production reliability.

### Recommendation

Build v1 in standard Go first, then evaluate TinyGo only if binary size/startup improvements are needed.

---

## 8. Dependencies

Minimum required runtime dependencies on host:

- `bluez` (Bluetooth stack and daemon)
- `dbus` (transitively required by most Linux BLE Go libraries)
- `systemd` (service management)

Go module dependencies are implementation-specific, but must support:

- BLE central scanning + GATT write/notify
- Linux/BlueZ compatibility on ARMv6/ARMv7 (Pi Zero W)

---

## 9. Error Handling and Retries

### Scan/connect/read failures

- Retry up to 3 attempts per poll cycle with backoff (`1s`, `2s`, `4s`).
- On failure, preserve prior cached reading (if any).
- Increment `consecutive_failures` and expose it in API response.

### Stale data behavior

- If cache exists but is stale, still return `200` with last known value and include age metadata in future revision.
- If no cache exists yet, return `503 pending`.

### Process resilience

- Fatal initialization failure exits non-zero.
- Runtime failures are logged and retried in next scheduled poll.
- `systemd` handles restart on crash.

---

## 10. Timing and Performance Targets

| Parameter | Default | Target |
|---|---|---|
| Poll interval | 30 minutes | Align with sensor update cadence |
| Scan timeout | 30 seconds | Find device in normal home BLE environment |
| Read timeout | 10 seconds | Trigger + receive notification |
| End-to-end poll cycle | <= 60 seconds | Includes scan, connect, read, disconnect |

---

## 11. Observability

### Logs

- Structured logs (JSON or key-value), including:
  - cycle start/end
  - selected device name/MAC/RSSI
  - decode result
  - retry attempts
  - failure reason

### Health

- `GET /healthz` for liveness.
- `GET /radon` for data freshness and failure counters.

---

## 12. Deployment

### Build (Go)

```sh
go build -o bin/radon-poller ./cmd/radon-poller
```

### Run locally

```sh
RADON_HTTP_ADDR=:8080 ./bin/radon-poller
```

### `systemd` unit (example)

```ini
[Unit]
Description=Radon Poller
After=network-online.target bluetooth.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=/opt/radon-poller
ExecStart=/opt/radon-poller/bin/radon-poller
Restart=always
RestartSec=5
Environment=RADON_POLL_INTERVAL=30m
Environment=RADON_HTTP_ADDR=:8080
Environment=RADON_NAME_PREFIX=FR:R20:

[Install]
WantedBy=multi-user.target
```

---

## 13. Implementation Phases

### Phase 1 — Baseline polling

- Implement BLE scan/connect/read/disconnect cycle.
- Decode pCi/L and convert to Bq/m^3.
- Verify against known R200 unit.

### Phase 2 — API + caching

- Add thread-safe cache.
- Add `/healthz` and `/radon`.
- Include pending/ok status behavior.

### Phase 3 — Production hardening

- Add systemd service packaging.
- Add metrics endpoint (`/metrics`) if needed.
- Add optional auth token and stale-age fields.
- Add integration tests with mocked BLE layer.

---

## 14. Risks and Assumptions

1. Linux BLE adapter permissions and BlueZ configuration may vary by distro image.
2. R200 advertisement naming convention is assumed stable (`FR:R20:`), but should remain configurable.
3. The sensor may reject concurrent/rapid polling; enforce one active poll cycle at a time.
4. Pi Zero W Bluetooth coexistence with WiFi is generally reliable under Linux, but local RF noise can increase scan latency.
