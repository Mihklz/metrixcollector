package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"

	models "github.com/Mihklz/metrixcollector/internal/model"
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

// SaveToFile сохраняет все метрики в файл в JSON формате
func (m *MemStorage) SaveToFile(filename string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var metrics []models.Metrics

	// Добавляем все gauge метрики
	for name, value := range m.Gauges {
		val := float64(value)
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: models.Gauge,
			Value: &val,
		})
	}

	// Добавляем все counter метрики
	for name, value := range m.Counters {
		val := int64(value)
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: models.Counter,
			Delta: &val,
		})
	}

	// Сериализуем в JSON с красивым форматированием
	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	// Записываем в файл
	err = os.WriteFile(filename, data, 0666)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// LoadFromFile загружает метрики из файла в JSON формате
func (m *MemStorage) LoadFromFile(filename string) error {
	// Проверяем, существует ли файл
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// Файл не существует - это нормально для первого запуска
		return nil
	}

	// Читаем файл
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Если файл пустой, ничего не делаем
	if len(data) == 0 {
		return nil
	}

	var metrics []models.Metrics
	err = json.Unmarshal(data, &metrics)
	if err != nil {
		return fmt.Errorf("failed to unmarshal metrics: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Загружаем метрики в storage
	for _, metric := range metrics {
		switch metric.MType {
		case models.Gauge:
			if metric.Value != nil {
				m.Gauges[metric.ID] = Gauge(*metric.Value)
			}
		case models.Counter:
			if metric.Delta != nil {
				m.Counters[metric.ID] = Counter(*metric.Delta)
			}
		}
	}

	return nil
}
