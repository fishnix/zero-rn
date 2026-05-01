package cache

import (
	"testing"
	"time"
)

func TestCacheInitialState(t *testing.T) {
	c := New()
	state := c.Get()
	if state.Valid {
		t.Fatalf("initial cache state should be invalid")
	}
	if state.ConsecutiveFailures != 0 {
		t.Fatalf("initial failures should be zero, got %d", state.ConsecutiveFailures)
	}
}

func TestCacheSetReadingResetsFailures(t *testing.T) {
	c := New()
	c.MarkFailure()
	c.MarkFailure()

	at := time.Now().UTC().Truncate(time.Second)
	c.SetReading(Reading{
		RadonPcIL:  1.63,
		RadonBqM3:  60.31,
		DeviceName: "FR:R20:SN0477",
		RSSI:       -49,
		MeasuredAt: at,
	})

	state := c.Get()
	if !state.Valid {
		t.Fatalf("cache should be valid after SetReading")
	}
	if state.ConsecutiveFailures != 0 {
		t.Fatalf("failures should reset to zero, got %d", state.ConsecutiveFailures)
	}
	if state.Reading.DeviceName != "FR:R20:SN0477" {
		t.Fatalf("unexpected device name: %s", state.Reading.DeviceName)
	}
}
