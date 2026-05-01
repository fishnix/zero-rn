package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("RADON_POLL_INTERVAL", "")
	t.Setenv("RADON_SCAN_TIMEOUT", "")
	t.Setenv("RADON_READ_TIMEOUT", "")
	t.Setenv("RADON_HTTP_ADDR", "")
	t.Setenv("RADON_NAME_PREFIX", "")
	t.Setenv("RADON_LOG_LEVEL", "")
	t.Setenv("RADON_DEVICE_MAC", "")
	t.Setenv("RADON_HTTP_TOKEN", "")

	cfg := Load()
	if got, want := cfg.PollInterval.String(), "30m0s"; got != want {
		t.Fatalf("PollInterval mismatch: got %s want %s", got, want)
	}
	if got, want := cfg.ScanTimeout.String(), "30s"; got != want {
		t.Fatalf("ScanTimeout mismatch: got %s want %s", got, want)
	}
	if got, want := cfg.ReadTimeout.String(), "10s"; got != want {
		t.Fatalf("ReadTimeout mismatch: got %s want %s", got, want)
	}
	if got, want := cfg.HTTPAddr, ":8080"; got != want {
		t.Fatalf("HTTPAddr mismatch: got %s want %s", got, want)
	}
	if got, want := cfg.NamePrefix, "FR:R20:"; got != want {
		t.Fatalf("NamePrefix mismatch: got %s want %s", got, want)
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	t.Setenv("RADON_POLL_INTERVAL", "5m")
	t.Setenv("RADON_SCAN_TIMEOUT", "9s")
	t.Setenv("RADON_READ_TIMEOUT", "4s")
	t.Setenv("RADON_HTTP_ADDR", ":9999")
	t.Setenv("RADON_NAME_PREFIX", "FR:TEST:")
	t.Setenv("RADON_LOG_LEVEL", "debug")
	t.Setenv("RADON_DEVICE_MAC", " AA:BB:CC:DD:EE:FF ")
	t.Setenv("RADON_HTTP_TOKEN", " token ")

	cfg := Load()
	if got, want := cfg.PollInterval.String(), "5m0s"; got != want {
		t.Fatalf("PollInterval mismatch: got %s want %s", got, want)
	}
	if got, want := cfg.ScanTimeout.String(), "9s"; got != want {
		t.Fatalf("ScanTimeout mismatch: got %s want %s", got, want)
	}
	if got, want := cfg.ReadTimeout.String(), "4s"; got != want {
		t.Fatalf("ReadTimeout mismatch: got %s want %s", got, want)
	}
	if got, want := cfg.HTTPAddr, ":9999"; got != want {
		t.Fatalf("HTTPAddr mismatch: got %s want %s", got, want)
	}
	if got, want := cfg.NamePrefix, "FR:TEST:"; got != want {
		t.Fatalf("NamePrefix mismatch: got %s want %s", got, want)
	}
	if got, want := cfg.LogLevel, "debug"; got != want {
		t.Fatalf("LogLevel mismatch: got %s want %s", got, want)
	}
	if got, want := cfg.DeviceMAC, "AA:BB:CC:DD:EE:FF"; got != want {
		t.Fatalf("DeviceMAC mismatch: got %s want %s", got, want)
	}
	if got, want := cfg.HTTPToken, "token"; got != want {
		t.Fatalf("HTTPToken mismatch: got %s want %s", got, want)
	}
}
