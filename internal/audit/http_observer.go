package audit

import (
	"bytes"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/Mihklz/metrixcollector/internal/logger"
)

// HTTPAuditObserver реализует Observer для отправки событий аудита по HTTP
type HTTPAuditObserver struct {
	url    string
	client *http.Client
}

// NewHTTPAuditObserver создает новый наблюдатель для отправки по HTTP
func NewHTTPAuditObserver(url string) *HTTPAuditObserver {
	return &HTTPAuditObserver{
		url: url,
		client: &http.Client{
			Timeout: 5 * time.Second, // Таймаут для HTTP-запросов
		},
	}
}

// Notify отправляет событие аудита на удаленный сервер по HTTP POST
func (h *HTTPAuditObserver) Notify(event *AuditEvent) error {
	// Преобразуем событие в JSON
	data, err := event.ToJSON()
	if err != nil {
		if logger.Log != nil {
			logger.Log.Error("Failed to marshal audit event to JSON",
				zap.Error(err),
			)
		}
		return err
	}

	// Создаем POST-запрос
	req, err := http.NewRequest(http.MethodPost, h.url, bytes.NewBuffer(data))
	if err != nil {
		if logger.Log != nil {
			logger.Log.Error("Failed to create HTTP request for audit",
				zap.String("url", h.url),
				zap.Error(err),
			)
		}
		return err
	}

	// Устанавливаем заголовок Content-Type
	req.Header.Set("Content-Type", "application/json")

	// Отправляем запрос
	resp, err := h.client.Do(req)
	if err != nil {
		if logger.Log != nil {
			logger.Log.Error("Failed to send audit event to HTTP endpoint",
				zap.String("url", h.url),
				zap.Error(err),
			)
		}
		return err
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if logger.Log != nil {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			logger.Log.Warn("HTTP audit endpoint returned non-success status",
				zap.String("url", h.url),
				zap.Int("status_code", resp.StatusCode),
			)
		} else {
			logger.Log.Debug("Audit event sent to HTTP endpoint",
				zap.String("url", h.url),
				zap.Int("status_code", resp.StatusCode),
				zap.Int64("timestamp", event.Timestamp),
			)
		}
	}

	return nil
}
