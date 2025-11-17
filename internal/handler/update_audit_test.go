package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Mihklz/metrixcollector/internal/audit"
	"github.com/Mihklz/metrixcollector/internal/repository"
)

// mockAuditObserver для тестирования аудита
type mockAuditObserver struct {
	events []*audit.AuditEvent
}

func (m *mockAuditObserver) Notify(event *audit.AuditEvent) error {
	m.events = append(m.events, event)
	return nil
}

// TestUpdateHandlerWithAudit проверяет интеграцию аудита с обработчиком update
func TestUpdateHandlerWithAudit(t *testing.T) {
	storage := repository.NewMemStorage()
	publisher := audit.NewAuditPublisher()
	observer := &mockAuditObserver{events: make([]*audit.AuditEvent, 0)}
	publisher.Subscribe(observer)

	handler := NewUpdateHandler(storage, publisher)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/TestMetric/42.5", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	// Даем время на асинхронную обработку аудита
	time.Sleep(100 * time.Millisecond)

	// Проверяем, что событие аудита было создано
	if len(observer.events) != 1 {
		t.Fatalf("Expected 1 audit event, got %d", len(observer.events))
	}

	event := observer.events[0]
	if len(event.Metrics) != 1 {
		t.Errorf("Expected 1 metric in audit event, got %d", len(event.Metrics))
	}

	if event.Metrics[0] != "TestMetric" {
		t.Errorf("Expected metric name TestMetric, got %s", event.Metrics[0])
	}

	if event.IPAddress != "192.168.1.100" {
		t.Errorf("Expected IP 192.168.1.100, got %s", event.IPAddress)
	}
}

// TestUpdateHandlerWithoutAudit проверяет работу без аудита
func TestUpdateHandlerWithoutAudit(t *testing.T) {
	storage := repository.NewMemStorage()

	// Передаем nil вместо publisher
	handler := NewUpdateHandler(storage, nil)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/TestMetric/42.5", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	// Не должно быть паники, запрос должен обработаться нормально
	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK even without audit, got %d", w.Code)
	}
}

// TestUpdateHandlerAuditOnlyOnSuccess проверяет, что аудит происходит только при успехе
func TestUpdateHandlerAuditOnlyOnSuccess(t *testing.T) {
	storage := repository.NewMemStorage()
	publisher := audit.NewAuditPublisher()
	observer := &mockAuditObserver{events: make([]*audit.AuditEvent, 0)}
	publisher.Subscribe(observer)

	handler := NewUpdateHandler(storage, publisher)

	// Отправляем невалидный запрос (некорректный тип метрики)
	req := httptest.NewRequest(http.MethodPost, "/update/invalid/TestMetric/value", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code == http.StatusOK {
		t.Error("Expected error status for invalid metric type")
	}

	time.Sleep(100 * time.Millisecond)

	// Проверяем, что событие аудита НЕ было создано
	if len(observer.events) != 0 {
		t.Errorf("Expected 0 audit events for failed request, got %d", len(observer.events))
	}
}

// TestUpdateHandlerAuditWithXForwardedFor проверяет извлечение IP из заголовков
func TestUpdateHandlerAuditWithXForwardedFor(t *testing.T) {
	storage := repository.NewMemStorage()
	publisher := audit.NewAuditPublisher()
	observer := &mockAuditObserver{events: make([]*audit.AuditEvent, 0)}
	publisher.Subscribe(observer)

	handler := NewUpdateHandler(storage, publisher)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/TestMetric/42.5", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.5, 198.51.100.10")
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
	// Должен использоваться первый IP из X-Forwarded-For
	if event.IPAddress != "203.0.113.5" {
		t.Errorf("Expected IP from X-Forwarded-For (203.0.113.5), got %s", event.IPAddress)
	}
}
