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
func (m *MemStorage) GetGauge(name string) (Gauge, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.Gauges[name]
	return val, ok
}

func (m *MemStorage) GetCounter(name string) (Counter, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.Counters[name]
	return val, ok
}

func (m *MemStorage) GetAllGauges() map[string]Gauge {
	m.mu.RLock()
	defer m.mu.RUnlock()

	copyMap := make(map[string]Gauge, len(m.Gauges))
	for k, v := range m.Gauges {
		copyMap[k] = v
	}
	return copyMap
}

func (m *MemStorage) GetAllCounters() map[string]Counter {
	m.mu.RLock()
	defer m.mu.RUnlock()

	copyMap := make(map[string]Counter, len(m.Counters))
	for k, v := range m.Counters {
		copyMap[k] = v
	}
	return copyMap
}
