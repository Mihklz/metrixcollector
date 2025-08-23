package handler

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"github.com/Mihklz/metrixcollector/internal/logger"
	models "github.com/Mihklz/metrixcollector/internal/model"
	"github.com/Mihklz/metrixcollector/internal/repository"
)

// NewJSONValueHandler создаёт обработчик для POST /value (JSON API)
// Принимает запрос с ID и типом метрики, возвращает её значение в JSON
func NewJSONValueHandler(storage repository.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверяем HTTP метод
		if r.Method != http.MethodPost {
			logger.Log.Info("Invalid method for JSON value",
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
		var request models.Metrics
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&request); err != nil {
			logger.Log.Info("Failed to decode JSON",
				zap.Error(err),
			)
			http.Error(w, "invalid JSON format", http.StatusBadRequest)
			return
		}

		// Валидируем обязательные поля
		if request.ID == "" {
			http.Error(w, "metric ID is required", http.StatusBadRequest)
			return
		}

		if request.MType == "" {
			http.Error(w, "metric type is required", http.StatusBadRequest)
			return
		}

		// Создаём ответ с теми же ID и MType
		response := models.Metrics{
			ID:    request.ID,
			MType: request.MType,
		}

		// Ищем метрику в зависимости от типа
		switch request.MType {
		case models.Gauge:
			if value, found := storage.GetGauge(request.ID); found {
				// Конвертируем в *float64 для JSON
				floatValue := float64(value)
				response.Value = &floatValue
			} else {
				http.Error(w, "metric not found", http.StatusNotFound)
				return
			}

		case models.Counter:
			if value, found := storage.GetCounter(request.ID); found {
				// Конвертируем в *int64 для JSON
				intValue := int64(value)
				response.Delta = &intValue
			} else {
				http.Error(w, "metric not found", http.StatusNotFound)
				return
			}

		default:
			http.Error(w, "unsupported metric type, expected 'gauge' or 'counter'", http.StatusBadRequest)
			return
		}

		// Логируем успешное получение
		logger.Log.Info("Metric retrieved successfully",
			zap.String("id", request.ID),
			zap.String("type", request.MType),
		)

		// Отправляем ответ в JSON
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		encoder := json.NewEncoder(w)
		if err := encoder.Encode(response); err != nil {
			logger.Log.Error("Failed to encode response JSON", zap.Error(err))
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
	}
}
