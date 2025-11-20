package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Mihklz/metrixcollector/internal/audit"
	"github.com/Mihklz/metrixcollector/internal/repository"
	"github.com/Mihklz/metrixcollector/internal/service"
)

// TestBatchUpdateHandlerWithAudit проверяет интеграцию аудита с batch API
func TestBatchUpdateHandlerWithAudit(t *testing.T) {
	storage := repository.NewMemStorage()
	metricsService := service.NewMetricsService(storage)
	publisher := audit.NewAuditPublisher()
	observer := &mockAuditObserver{events: make([]*audit.AuditEvent, 0)}
	publisher.Subscribe(observer)

	handler := NewBatchUpdateHandler(metricsService, "", publisher)

	jsonBody := `[
		{"id":"Metric1","type":"gauge","value":123.45},
		{"id":"Metric2","type":"counter","delta":100},
		{"id":"Metric3","type":"gauge","value":67.89}
	]`

	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewBufferString(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "10.20.30.40:12345"
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	time.Sleep(150 * time.Millisecond)

	// Проверяем, что создано одно событие аудита с тремя метриками
	if len(observer.events) != 1 {
		t.Fatalf("Expected 1 audit event, got %d", len(observer.events))
	}

	event := observer.events[0]
	if len(event.Metrics) != 3 {
		t.Errorf("Expected 3 metrics in audit event, got %d", len(event.Metrics))
	}

	// Проверяем наличие всех метрик
	expectedMetrics := map[string]bool{
		"Metric1": false,
		"Metric2": false,
		"Metric3": false,
	}

	for _, metric := range event.Metrics {
		if _, exists := expectedMetrics[metric]; exists {
			expectedMetrics[metric] = true
		}
	}

	for metric, found := range expectedMetrics {
		if !found {
			t.Errorf("Metric %s not found in audit event", metric)
		}
	}

	if event.IPAddress != "10.20.30.40" {
		t.Errorf("Expected IP 10.20.30.40, got %s", event.IPAddress)
	}
}

// TestBatchUpdateHandlerWithoutAudit проверяет работу без аудита
func TestBatchUpdateHandlerWithoutAudit(t *testing.T) {
	storage := repository.NewMemStorage()
	metricsService := service.NewMetricsService(storage)
	handler := NewBatchUpdateHandler(metricsService, "", nil)

	jsonBody := `[{"id":"Metric1","type":"gauge","value":1.0}]`

	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewBufferString(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK even without audit, got %d", w.Code)
	}
}

// TestBatchUpdateHandlerAuditOnlyOnSuccess проверяет аудит только при успехе
func TestBatchUpdateHandlerAuditOnlyOnSuccess(t *testing.T) {
	storage := repository.NewMemStorage()
	metricsService := service.NewMetricsService(storage)
	publisher := audit.NewAuditPublisher()
	observer := &mockAuditObserver{events: make([]*audit.AuditEvent, 0)}
	publisher.Subscribe(observer)

	handler := NewBatchUpdateHandler(metricsService, "", publisher)

	// Невалидный JSON
	jsonBody := `[{"invalid": "data"`

	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewBufferString(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code == http.StatusOK {
		t.Error("Expected error status for invalid JSON")
	}

	time.Sleep(100 * time.Millisecond)

	if len(observer.events) != 0 {
		t.Errorf("Expected 0 audit events for failed request, got %d", len(observer.events))
	}
}

// TestBatchUpdateHandlerEmptyBatch проверяет пустой батч
func TestBatchUpdateHandlerEmptyBatch(t *testing.T) {
	storage := repository.NewMemStorage()
	metricsService := service.NewMetricsService(storage)
	publisher := audit.NewAuditPublisher()
	observer := &mockAuditObserver{events: make([]*audit.AuditEvent, 0)}
	publisher.Subscribe(observer)

	handler := NewBatchUpdateHandler(metricsService, "", publisher)

	jsonBody := `[]`

	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewBufferString(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	// Пустой батч считается ошибкой валидации
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest for empty batch, got %d", w.Code)
	}

	time.Sleep(100 * time.Millisecond)

	// Для неуспешных запросов событие аудита не создается
	if len(observer.events) != 0 {
		t.Errorf("Expected 0 audit events for empty batch, got %d", len(observer.events))
	}
}

// TestBatchUpdateHandlerLargeBatch проверяет большой батч
func TestBatchUpdateHandlerLargeBatch(t *testing.T) {
	storage := repository.NewMemStorage()
	metricsService := service.NewMetricsService(storage)
	publisher := audit.NewAuditPublisher()
	observer := &mockAuditObserver{events: make([]*audit.AuditEvent, 0)}
	publisher.Subscribe(observer)

	handler := NewBatchUpdateHandler(metricsService, "", publisher)

	// Создаем батч с 100 метриками
	var jsonBody bytes.Buffer
	jsonBody.WriteString("[")
	for i := 0; i < 100; i++ {
		if i > 0 {
			jsonBody.WriteString(",")
		}
		jsonBody.WriteString(`{"id":"Metric`)
		jsonBody.WriteString(string(rune('0' + i%10)))
		jsonBody.WriteString(`","type":"gauge","value":`)
		jsonBody.WriteString(string(rune('0' + i%10)))
		jsonBody.WriteString(`.0}`)
	}
	jsonBody.WriteString("]")

	req := httptest.NewRequest(http.MethodPost, "/updates/", &jsonBody)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK for large batch, got %d", w.Code)
	}

	time.Sleep(150 * time.Millisecond)

	if len(observer.events) != 1 {
		t.Fatalf("Expected 1 audit event, got %d", len(observer.events))
	}

	event := observer.events[0]
	if len(event.Metrics) != 100 {
		t.Errorf("Expected 100 metrics in audit event, got %d", len(event.Metrics))
	}
}
