package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"github.com/Mihklz/metrixcollector/internal/logger"
	models "github.com/Mihklz/metrixcollector/internal/model"
	"github.com/Mihklz/metrixcollector/internal/repository"
)

// NewJSONUpdateHandler создаёт обработчик для POST /update (JSON API)
// Принимает метрики в формате JSON и сохраняет их в хранилище
func NewJSONUpdateHandler(storage repository.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверяем HTTP метод
		if r.Method != http.MethodPost {
			logger.Log.Info("Invalid method for JSON update",
				zap.String("method", r.Method),
				zap.String("expected", http.MethodPost),
			)
			http.Error(w, "only POST method allowed", http.StatusMethodNotAllowed)
			return
		}

		// Проверяем Content-Type
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			logger.Log.Info("Invalid content type",
				zap.String("content_type", contentType),
				zap.String("expected", "application/json"),
			)
			http.Error(w, "content type must be application/json", http.StatusBadRequest)
			return
		}

		// Декодируем JSON из тела запроса
		var metric models.Metrics
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&metric); err != nil {
			logger.Log.Info("Failed to decode JSON",
				zap.Error(err),
			)
			http.Error(w, "invalid JSON format", http.StatusBadRequest)
			return
		}

		// Валидируем обязательные поля
		if metric.ID == "" {
			http.Error(w, "metric ID is required", http.StatusBadRequest)
			return
		}

		if metric.MType == "" {
			http.Error(w, "metric type is required", http.StatusBadRequest)
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

		// Отправляем успешный ответ
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Возвращаем сохранённую метрику в ответе
		encoder := json.NewEncoder(w)
		if err := encoder.Encode(metric); err != nil {
			logger.Log.Error("Failed to encode response JSON", zap.Error(err))
		}
	}
}
