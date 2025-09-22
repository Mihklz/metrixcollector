package handler

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"github.com/Mihklz/metrixcollector/internal/logger"
	models "github.com/Mihklz/metrixcollector/internal/model"
	"github.com/Mihklz/metrixcollector/internal/service"
)

// BatchUpdateHandler обрабатывает запросы для пакетного обновления метрик
type BatchUpdateHandler struct {
	metricsService *service.MetricsService
}

// NewBatchUpdateHandler создает новый обработчик для пакетного обновления метрик
func NewBatchUpdateHandler(metricsService *service.MetricsService) http.HandlerFunc {
	handler := &BatchUpdateHandler{metricsService: metricsService}
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

	// Используем сервис для обновления метрик
	if err := h.metricsService.UpdateBatch(metrics); err != nil {
		// Проверяем тип ошибки - если это ошибка валидации, возвращаем 400
		if service.IsValidationError(err) {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}
