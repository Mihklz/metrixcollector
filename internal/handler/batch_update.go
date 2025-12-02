package handler

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"github.com/Mihklz/metrixcollector/internal/audit"
	"github.com/Mihklz/metrixcollector/internal/logger"
	models "github.com/Mihklz/metrixcollector/internal/model"
	"github.com/Mihklz/metrixcollector/internal/service"
)

// BatchUpdateHandler обрабатывает запросы для пакетного обновления метрик.
type BatchUpdateHandler struct {
	metricsService *service.MetricsService
	key            string
	auditPublisher *audit.AuditPublisher
}

// NewBatchUpdateHandler создает новый обработчик для пакетного обновления метрик.
func NewBatchUpdateHandler(metricsService *service.MetricsService, key string, auditPublisher *audit.AuditPublisher) http.HandlerFunc {
	handler := &BatchUpdateHandler{
		metricsService: metricsService,
		key:            key,
		auditPublisher: auditPublisher,
	}
	return handler.Handle
}

// Handle обрабатывает POST запрос к /updates/.
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

	// Публикуем событие аудита после успешной обработки
	if h.auditPublisher != nil && h.auditPublisher.HasObservers() {
		// Собираем имена всех метрик
		metricNames := make([]string, 0, len(metrics))
		for _, m := range metrics {
			metricNames = append(metricNames, m.ID)
		}
		event := audit.NewAuditEvent(metricNames, audit.GetIPAddress(r))
		h.auditPublisher.Publish(event)
	}

	// Отправляем пустой ответ с хешем
	WriteResponseWithHash(w, []byte(""), h.key, http.StatusOK, "application/json")
}
