package radoneye

import (
	"errors"
	"math"
	"testing"
)

func TestDecodeRadonPcIL(t *testing.T) {
	want := float32(1.63)
	bits := math.Float32bits(want)
	payload := []byte{
		0x50,
		0x00,
		byte(bits),
		byte(bits >> 8),
		byte(bits >> 16),
		byte(bits >> 24),
	}

	got, err := decodeRadonPcIL(payload)
	if err != nil {
		t.Fatalf("decodeRadonPcIL returned error: %v", err)
	}
	if math.Abs(float64(got-want)) > 0.00001 {
		t.Fatalf("decodeRadonPcIL mismatch: got %v want %v", got, want)
	}
}

func TestDecodeRadonPcILShortPayload(t *testing.T) {
	_, err := decodeRadonPcIL([]byte{0x00, 0x01, 0x02})
	if !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("expected ErrInvalidPayload, got %v", err)
	}
}
