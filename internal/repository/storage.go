package repository

import models "github.com/Mihklz/metrixcollector/internal/model"

type Storage interface {
	Update(metricType, name, value string) error
	GetGauge(name string) (Gauge, bool)
	GetCounter(name string) (Counter, bool)
	GetAllGauges() map[string]Gauge
	GetAllCounters() map[string]Counter
	// Новые методы для файлового хранения
	SaveToFile(filename string) error
	LoadFromFile(filename string) error
}

// BatchStorage интерфейс для пакетных операций с метриками
type BatchStorage interface {
	Storage
	// UpdateBatch обновляет множество метрик в рамках одной транзакции
	UpdateBatch(metrics []models.Metrics) error
}
