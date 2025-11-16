package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"github.com/Mihklz/metrixcollector/internal/audit"
	"github.com/Mihklz/metrixcollector/internal/logger"
	models "github.com/Mihklz/metrixcollector/internal/model"
	"github.com/Mihklz/metrixcollector/internal/repository"
)

// validateJSONRequest проверяет HTTP метод, Content-Type и декодирует JSON
func validateJSONRequest(w http.ResponseWriter, r *http.Request) (*models.Metrics, bool) {
	// Проверяем HTTP метод
	if r.Method != http.MethodPost {
		logger.Log.Info("Invalid method for JSON API",
			zap.String("method", r.Method),
			zap.String("expected", http.MethodPost),
		)
		http.Error(w, "only POST method allowed", http.StatusMethodNotAllowed)
		return nil, false
	}

	// Проверяем Content-Type
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		logger.Log.Info("Invalid content type",
			zap.String("content_type", contentType),
			zap.String("expected", "application/json"),
		)
		http.Error(w, "content type must be application/json", http.StatusBadRequest)
		return nil, false
	}

	// Декодируем JSON из тела запроса
	var metric models.Metrics
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&metric); err != nil {
		logger.Log.Info("Failed to decode JSON", zap.Error(err))
		http.Error(w, "invalid JSON format", http.StatusBadRequest)
		return nil, false
	}

	// Валидируем обязательные поля
	if metric.ID == "" {
		http.Error(w, "metric ID is required", http.StatusBadRequest)
		return nil, false
	}

	if metric.MType == "" {
		http.Error(w, "metric type is required", http.StatusBadRequest)
		return nil, false
	}

	return &metric, true
}

// NewJSONUpdateHandler создаёт обработчик для POST /update (JSON API)
// Принимает метрики в формате JSON и сохраняет их в хранилище
func NewJSONUpdateHandler(storage repository.Storage, key string, auditPublisher *audit.AuditPublisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Общая валидация и декодирование
		metric, ok := validateJSONRequest(w, r)
		if !ok {
			return
		}

		// Валидируем тип метрики и наличие соответствующего значения
		var valueStr string
		switch metric.MType {
		case models.Gauge:
			if metric.Value == nil {
				http.Error(w, "value is required for gauge metric", http.StatusBadRequest)
				return
			}
			// Конвертируем float64 в строку для storage.Update
			valueStr = fmt.Sprintf("%g", *metric.Value)

		case models.Counter:
			if metric.Delta == nil {
				http.Error(w, "delta is required for counter metric", http.StatusBadRequest)
				return
			}
			// Конвертируем int64 в строку для storage.Update
			valueStr = fmt.Sprintf("%d", *metric.Delta)

		default:
			http.Error(w, "unsupported metric type, expected 'gauge' or 'counter'", http.StatusBadRequest)
			return
		}

		// Сохраняем метрику через существующий интерфейс
		err := storage.Update(metric.MType, metric.ID, valueStr)
		if err != nil {
			logger.Log.Error("Failed to save metric",
				zap.String("id", metric.ID),
				zap.String("type", metric.MType),
				zap.String("value", valueStr),
				zap.Error(err),
			)
			http.Error(w, "failed to save metric", http.StatusInternalServerError)
			return
		}

	// Логируем успешное сохранение
	logger.Log.Info("Metric saved successfully",
		zap.String("id", metric.ID),
		zap.String("type", metric.MType),
	)

	// Публикуем событие аудита после успешной обработки
	if auditPublisher != nil && auditPublisher.HasObservers() {
		event := audit.NewAuditEvent([]string{metric.ID}, audit.GetIPAddress(r))
		auditPublisher.Publish(event)
	}

	// Возвращаем сохранённую метрику в ответе с хешем
	responseData, err := json.Marshal(metric)
		if err != nil {
			logger.Log.Error("Failed to encode response JSON", zap.Error(err))
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
			return
		}

		WriteResponseWithHash(w, responseData, key, http.StatusOK, "application/json")
	}
}
