package audit

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestHTTPAuditObserver проверяет отправку событий по HTTP
func TestHTTPAuditObserver(t *testing.T) {
	// Создаем тестовый HTTP сервер
	receivedEvents := make([]*AuditEvent, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем метод
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Проверяем Content-Type
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}

		// Читаем и декодируем тело запроса
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var event AuditEvent
		if err := json.Unmarshal(body, &event); err != nil {
			t.Errorf("Failed to unmarshal event: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		receivedEvents = append(receivedEvents, &event)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Создаем HTTP наблюдателя
	observer := NewHTTPAuditObserver(server.URL)
	event := NewAuditEvent([]string{"Metric1", "Metric2"}, "10.0.0.5")

	// Отправляем событие
	err := observer.Notify(event)
	if err != nil {
		t.Fatalf("Failed to notify HTTP observer: %v", err)
	}

	// Проверяем, что событие получено
	if len(receivedEvents) != 1 {
		t.Fatalf("Expected 1 received event, got %d", len(receivedEvents))
	}

	received := receivedEvents[0]
	if len(received.Metrics) != 2 {
		t.Errorf("Expected 2 metrics, got %d", len(received.Metrics))
	}

	if received.IPAddress != "10.0.0.5" {
		t.Errorf("Expected IP 10.0.0.5, got %s", received.IPAddress)
	}
}

// TestHTTPAuditObserverNonSuccessStatus проверяет обработку не-успешных статусов
func TestHTTPAuditObserverNonSuccessStatus(t *testing.T) {
	// Создаем сервер, который возвращает ошибку
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	observer := NewHTTPAuditObserver(server.URL)
	event := NewAuditEvent([]string{"TestMetric"}, "127.0.0.1")

	// Событие должно быть отправлено, но может залогировать предупреждение
	err := observer.Notify(event)
	if err != nil {
		t.Errorf("Notify should not return error for non-success status, got: %v", err)
	}
}

// TestHTTPAuditObserverInvalidURL проверяет обработку невалидного URL
func TestHTTPAuditObserverInvalidURL(t *testing.T) {
	observer := NewHTTPAuditObserver("http://invalid-host-that-does-not-exist-12345.com")
	event := NewAuditEvent([]string{"TestMetric"}, "127.0.0.1")

	// Ожидаем ошибку при попытке отправки
	err := observer.Notify(event)
	if err == nil {
		t.Error("Expected error for invalid URL, got nil")
	}
}

// TestHTTPAuditObserverTimeout проверяет таймаут
func TestHTTPAuditObserverTimeout(t *testing.T) {
	// Создаем сервер, который долго отвечает
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second) // Больше, чем таймаут наблюдателя (5 секунд)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	observer := NewHTTPAuditObserver(server.URL)
	event := NewAuditEvent([]string{"TestMetric"}, "127.0.0.1")

	// Ожидаем ошибку таймаута
	start := time.Now()
	err := observer.Notify(event)
	duration := time.Since(start)

	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	// Проверяем, что таймаут сработал примерно через 5 секунд
	if duration < 4*time.Second || duration > 7*time.Second {
		t.Errorf("Expected timeout around 5 seconds, got %v", duration)
	}
}

// TestHTTPAuditObserverConcurrent проверяет конкурентную отправку событий
func TestHTTPAuditObserverConcurrent(t *testing.T) {
	receivedCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	observer := NewHTTPAuditObserver(server.URL)

	// Отправляем 10 событий конкурентно
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			event := NewAuditEvent([]string{"Metric" + string(rune(idx))}, "127.0.0.1")
			observer.Notify(event)
			done <- true
		}(i)
	}

	// Ждем завершения всех горутин
	for i := 0; i < 10; i++ {
		<-done
	}

	// Даем время на обработку
	time.Sleep(100 * time.Millisecond)

	if receivedCount != 10 {
		t.Errorf("Expected 10 received events, got %d", receivedCount)
	}
}
