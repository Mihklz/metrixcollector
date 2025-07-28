package repository

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
)

type Gauge float64
type Counter int64

type MemStorage struct {
	mu       sync.RWMutex
	Gauges   map[string]Gauge
	Counters map[string]Counter
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		Gauges:   make(map[string]Gauge),
		Counters: make(map[string]Counter),
	}
}

func (m *MemStorage) Update(metricType, name, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch metricType {
	case "gauge":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid gauge value: %w", err)
		}
		m.Gauges[name] = Gauge(v)
	case "counter":
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid counter value: %w", err)
		}
		m.Counters[name] += Counter(v)
	default:
		return errors.New("unsupported metric type")
	}

	return nil
}
