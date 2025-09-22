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
func NewJSONValueHandler(storage repository.Storage, key string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Общая валидация и декодирование
		request, ok := validateJSONRequest(w, r)
		if !ok {
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

		// Отправляем ответ в JSON с хешем
		responseData, err := json.Marshal(response)
		if err != nil {
			logger.Log.Error("Failed to encode response JSON", zap.Error(err))
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
			return
		}

		WriteResponseWithHash(w, responseData, key, http.StatusOK, "application/json")
	}
}
