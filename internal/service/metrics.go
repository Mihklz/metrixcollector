package service

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/Mihklz/metrixcollector/internal/logger"
	models "github.com/Mihklz/metrixcollector/internal/model"
	"github.com/Mihklz/metrixcollector/internal/repository"
)

// MetricsService отвечает за бизнес-логику работы с метриками.
type MetricsService struct {
	storage repository.Storage
	logger  *zap.Logger
}

// NewMetricsService создает новый сервис метрик.
func NewMetricsService(storage repository.Storage) *MetricsService {
	return &MetricsService{
		storage: storage,
		logger:  logger.Log,
	}
}

// ValidationError представляет ошибку валидации.
type ValidationError struct {
	Message string
}

// Error возвращает текст ошибки валидации.
func (e *ValidationError) Error() string {
	return e.Message
}

// IsValidationError проверяет, является ли ошибка ошибкой валидации.
func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}

// validateMetric проверяет корректность метрики
func (s *MetricsService) validateMetric(metric models.Metrics) error {
	switch metric.MType {
	case models.Counter:
		if metric.Delta == nil {
			return &ValidationError{Message: fmt.Sprintf("Counter metric '%s' missing delta", metric.ID)}
		}
	case models.Gauge:
		if metric.Value == nil {
			return &ValidationError{Message: fmt.Sprintf("Gauge metric '%s' missing value", metric.ID)}
		}
	default:
		return &ValidationError{Message: fmt.Sprintf("Unknown metric type '%s' for metric '%s'", metric.MType, metric.ID)}
	}
	return nil
}

// UpdateBatch обновляет множество метрик с валидацией.
func (s *MetricsService) UpdateBatch(metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return &ValidationError{Message: "Batch cannot be empty"}
	}

	// Валидируем все метрики перед обновлением
	for _, metric := range metrics {
		if err := s.validateMetric(metric); err != nil {
			s.logger.Error("Metric validation failed",
				zap.String("metric_id", metric.ID),
				zap.String("metric_type", metric.MType),
				zap.Error(err))
			return err
		}
	}

	// Используем batch операцию, если хранилище поддерживает её
	if batchStorage, ok := s.storage.(repository.BatchStorage); ok {
		if err := batchStorage.UpdateBatch(metrics); err != nil {
			s.logger.Error("Failed to update metrics batch", zap.Error(err))
			return fmt.Errorf("failed to update metrics batch: %w", err)
		}
		s.logger.Info("Batch metrics updated successfully", zap.Int("count", len(metrics)))
		return nil
	}

	// Fallback на обычные операции для обратной совместимости
	for _, metric := range metrics {
		var value string

		switch metric.MType {
		case models.Counter:
			value = fmt.Sprintf("%d", *metric.Delta)
		case models.Gauge:
			value = fmt.Sprintf("%g", *metric.Value)
		}

		if err := s.storage.Update(metric.MType, metric.ID, value); err != nil {
			s.logger.Error("Failed to update metric",
				zap.Error(err),
				zap.String("type", metric.MType),
				zap.String("id", metric.ID),
			)
			return fmt.Errorf("failed to update metric %s: %w", metric.ID, err)
		}
	}

	s.logger.Info("Batch metrics updated successfully", zap.Int("count", len(metrics)))
	return nil
}

// UpdateSingle обновляет одну метрику с валидацией.
func (s *MetricsService) UpdateSingle(metric models.Metrics) error {
	if err := s.validateMetric(metric); err != nil {
		s.logger.Error("Metric validation failed",
			zap.String("metric_id", metric.ID),
			zap.String("metric_type", metric.MType),
			zap.Error(err))
		return err
	}

	var value string
	switch metric.MType {
	case models.Counter:
		value = fmt.Sprintf("%d", *metric.Delta)
	case models.Gauge:
		value = fmt.Sprintf("%g", *metric.Value)
	}

	if err := s.storage.Update(metric.MType, metric.ID, value); err != nil {
		s.logger.Error("Failed to update metric",
			zap.Error(err),
			zap.String("type", metric.MType),
			zap.String("id", metric.ID),
		)
		return fmt.Errorf("failed to update metric %s: %w", metric.ID, err)
	}

	s.logger.Info("Metric updated successfully",
		zap.String("type", metric.MType),
		zap.String("id", metric.ID))
	return nil
}
