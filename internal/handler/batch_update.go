package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/Mihklz/metrixcollector/internal/logger"
	models "github.com/Mihklz/metrixcollector/internal/model"
	"github.com/Mihklz/metrixcollector/internal/repository"
)

// BatchUpdateHandler обрабатывает запросы для пакетного обновления метрик
type BatchUpdateHandler struct {
	storage repository.Storage
}

// NewBatchUpdateHandler создает новый обработчик для пакетного обновления метрик
func NewBatchUpdateHandler(storage repository.Storage) http.HandlerFunc {
	handler := &BatchUpdateHandler{storage: storage}
	return handler.Handle
}

// Handle обрабатывает POST запрос к /updates/
func (h *BatchUpdateHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Проверяем Content-Type
	if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var metrics []models.Metrics
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&metrics); err != nil {
		logger.Log.Error("Failed to decode batch metrics request", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Проверяем, что батч не пустой
	if len(metrics) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Используем batch операцию, если хранилище поддерживает её
	if batchStorage, ok := h.storage.(repository.BatchStorage); ok {
		if err := batchStorage.UpdateBatch(metrics); err != nil {
			logger.Log.Error("Failed to update metrics batch", zap.Error(err))

			// Проверяем тип ошибки - если это ошибка валидации, возвращаем 400
			if isValidationError(err) {
				w.WriteHeader(http.StatusBadRequest)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}
	} else {
		// Fallback на обычные операции для обратной совместимости
		for _, metric := range metrics {
			var value string

			switch metric.MType {
			case models.Counter:
				if metric.Delta == nil {
					logger.Log.Error("Counter metric missing delta", zap.String("id", metric.ID))
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				value = fmt.Sprintf("%d", *metric.Delta)
			case models.Gauge:
				if metric.Value == nil {
					logger.Log.Error("Gauge metric missing value", zap.String("id", metric.ID))
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				value = fmt.Sprintf("%g", *metric.Value)
			default:
				logger.Log.Error("Unknown metric type", zap.String("type", metric.MType), zap.String("id", metric.ID))
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if err := h.storage.Update(metric.MType, metric.ID, value); err != nil {
				logger.Log.Error("Failed to update metric",
					zap.Error(err),
					zap.String("type", metric.MType),
					zap.String("id", metric.ID),
				)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}

	logger.Log.Info("Batch metrics updated successfully", zap.Int("count", len(metrics)))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

// isValidationError проверяет, является ли ошибка ошибкой валидации
func isValidationError(err error) bool {
	errMsg := err.Error()
	return strings.Contains(errMsg, "missing value") ||
		strings.Contains(errMsg, "missing delta") ||
		strings.Contains(errMsg, "unsupported metric type")
}
