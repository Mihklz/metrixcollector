package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Mihklz/metrixcollector/internal/audit"
	"github.com/Mihklz/metrixcollector/internal/repository"
)

// TestJSONUpdateHandlerWithAudit проверяет интеграцию аудита с JSON API
func TestJSONUpdateHandlerWithAudit(t *testing.T) {
	storage := repository.NewMemStorage()
	publisher := audit.NewAuditPublisher()
	observer := &mockAuditObserver{events: make([]*audit.AuditEvent, 0)}
	publisher.Subscribe(observer)

	handler := NewJSONUpdateHandler(storage, "", publisher)

	jsonBody := `{"id":"TestCounter","type":"counter","delta":100}`
	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBufferString(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.1.50:12345"
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	time.Sleep(100 * time.Millisecond)

	if len(observer.events) != 1 {
		t.Fatalf("Expected 1 audit event, got %d", len(observer.events))
	}

	event := observer.events[0]
	if len(event.Metrics) != 1 {
		t.Errorf("Expected 1 metric, got %d", len(event.Metrics))
	}

	if event.Metrics[0] != "TestCounter" {
		t.Errorf("Expected metric TestCounter, got %s", event.Metrics[0])
	}

	if event.IPAddress != "192.168.1.50" {
		t.Errorf("Expected IP 192.168.1.50, got %s", event.IPAddress)
	}
}

// TestJSONUpdateHandlerWithoutAudit проверяет работу без аудита
func TestJSONUpdateHandlerWithoutAudit(t *testing.T) {
	storage := repository.NewMemStorage()
	handler := NewJSONUpdateHandler(storage, "", nil)

	jsonBody := `{"id":"TestGauge","type":"gauge","value":42.5}`
	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBufferString(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK even without audit, got %d", w.Code)
	}
}

// TestJSONUpdateHandlerAuditOnlyOnSuccess проверяет аудит только при успехе
func TestJSONUpdateHandlerAuditOnlyOnSuccess(t *testing.T) {
	storage := repository.NewMemStorage()
	publisher := audit.NewAuditPublisher()
	observer := &mockAuditObserver{events: make([]*audit.AuditEvent, 0)}
	publisher.Subscribe(observer)

	handler := NewJSONUpdateHandler(storage, "", publisher)

	// Невалидный JSON
	jsonBody := `{"invalid json`
	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBufferString(jsonBody))
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

// TestJSONUpdateHandlerAuditMultipleRequests проверяет несколько запросов
func TestJSONUpdateHandlerAuditMultipleRequests(t *testing.T) {
	storage := repository.NewMemStorage()
	publisher := audit.NewAuditPublisher()
	observer := &mockAuditObserver{events: make([]*audit.AuditEvent, 0)}
	publisher.Subscribe(observer)

	handler := NewJSONUpdateHandler(storage, "", publisher)

	// Отправляем несколько запросов
	requests := []string{
		`{"id":"Metric1","type":"gauge","value":1.0}`,
		`{"id":"Metric2","type":"gauge","value":2.0}`,
		`{"id":"Metric3","type":"counter","delta":3}`,
	}

	for _, jsonBody := range requests {
		req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBufferString(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status OK, got %d for body: %s", w.Code, jsonBody)
		}
	}

	time.Sleep(200 * time.Millisecond)

	if len(observer.events) != 3 {
		t.Errorf("Expected 3 audit events, got %d", len(observer.events))
	}

	// Проверяем, что все метрики зафиксированы
	expectedMetrics := map[string]bool{"Metric1": false, "Metric2": false, "Metric3": false}
	for _, event := range observer.events {
		if len(event.Metrics) != 1 {
			continue
		}
		metricName := event.Metrics[0]
		if _, exists := expectedMetrics[metricName]; exists {
			expectedMetrics[metricName] = true
		}
	}

	for metric, found := range expectedMetrics {
		if !found {
			t.Errorf("Metric %s not found in audit events", metric)
		}
	}
}
