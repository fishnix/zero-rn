package radoneye

import (
	"encoding/binary"
	"errors"
	"math"
)

var ErrInvalidPayload = errors.New("invalid radoneye payload length")

func decodeRadonPcIL(payload []byte) (float32, error) {
	if len(payload) < 6 {
		return 0, ErrInvalidPayload
	}
	bits := binary.LittleEndian.Uint32(payload[2:6])
	return math.Float32frombits(bits), nil
}
