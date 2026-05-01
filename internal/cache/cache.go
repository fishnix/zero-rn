package cache

import (
	"sync"
	"time"
)

type Reading struct {
	RadonPcIL  float32   `json:"radon_pci_l"`
	RadonBqM3  float32   `json:"radon_bq_m3"`
	DeviceName string    `json:"device_name"`
	RSSI       int16     `json:"rssi_dbm"`
	MeasuredAt time.Time `json:"measured_at"`
}

type State struct {
	Reading             Reading
	Valid               bool
	ConsecutiveFailures int
}

type Cache struct {
	mu    sync.RWMutex
	state State
}

func New() *Cache {
	return &Cache{}
}

func (c *Cache) SetReading(reading Reading) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.Reading = reading
	c.state.Valid = true
	c.state.ConsecutiveFailures = 0
}

func (c *Cache) MarkFailure() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.ConsecutiveFailures++
}

func (c *Cache) Get() State {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}
