package audit

import (
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// TestNewAuditEvent проверяет создание события аудита
func TestNewAuditEvent(t *testing.T) {
	metrics := []string{"Alloc", "Frees", "NumGC"}
	ipAddress := "192.168.1.1"

	event := NewAuditEvent(metrics, ipAddress)

	if event == nil {
		t.Fatal("NewAuditEvent returned nil")
	}

	if event.IPAddress != ipAddress {
		t.Errorf("Expected IP address %s, got %s", ipAddress, event.IPAddress)
	}

	if len(event.Metrics) != len(metrics) {
		t.Errorf("Expected %d metrics, got %d", len(metrics), len(event.Metrics))
	}

	for i, metric := range metrics {
		if event.Metrics[i] != metric {
			t.Errorf("Expected metric %s at index %d, got %s", metric, i, event.Metrics[i])
		}
	}

	if event.Timestamp == 0 {
		t.Error("Timestamp should not be zero")
	}

	// Проверяем, что timestamp близок к текущему времени (в пределах 1 секунды)
	now := time.Now().Unix()
	if event.Timestamp < now-1 || event.Timestamp > now+1 {
		t.Errorf("Timestamp %d is not close to current time %d", event.Timestamp, now)
	}
}

// TestAuditEventToJSON проверяет сериализацию события в JSON
func TestAuditEventToJSON(t *testing.T) {
	event := &AuditEvent{
		Timestamp: 1234567890,
		Metrics:   []string{"Metric1", "Metric2"},
		IPAddress: "10.0.0.1",
	}

	jsonData, err := event.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert event to JSON: %v", err)
	}

	jsonStr := string(jsonData)

	// Проверяем наличие всех полей в JSON
	if !strings.Contains(jsonStr, `"ts":1234567890`) {
		t.Error("JSON does not contain correct timestamp")
	}

	if !strings.Contains(jsonStr, `"metrics":["Metric1","Metric2"]`) {
		t.Error("JSON does not contain correct metrics")
	}

	if !strings.Contains(jsonStr, `"ip_address":"10.0.0.1"`) {
		t.Error("JSON does not contain correct IP address")
	}
}

// TestAuditPublisherSubscribe проверяет подписку наблюдателей
func TestAuditPublisherSubscribe(t *testing.T) {
	publisher := NewAuditPublisher()

	if publisher.HasObservers() {
		t.Error("Publisher should not have observers initially")
	}

	// Создаем mock наблюдателя
	mockObserver := &mockObserver{}
	publisher.Subscribe(mockObserver)

	if !publisher.HasObservers() {
		t.Error("Publisher should have observers after subscription")
	}
}

// TestAuditPublisherUnsubscribe проверяет отписку наблюдателей
func TestAuditPublisherUnsubscribe(t *testing.T) {
	publisher := NewAuditPublisher()
	mockObserver := &mockObserver{}

	publisher.Subscribe(mockObserver)
	if !publisher.HasObservers() {
		t.Error("Publisher should have observers after subscription")
	}

	publisher.Unsubscribe(mockObserver)
	if publisher.HasObservers() {
		t.Error("Publisher should not have observers after unsubscription")
	}
}

// TestAuditPublisherPublish проверяет рассылку событий
func TestAuditPublisherPublish(t *testing.T) {
	publisher := NewAuditPublisher()
	mockObserver := &mockObserver{
		events: make([]*AuditEvent, 0),
	}

	publisher.Subscribe(mockObserver)

	event := NewAuditEvent([]string{"TestMetric"}, "127.0.0.1")
	publisher.Publish(event)

	// Даем время на обработку события (асинхронная отправка)
	time.Sleep(100 * time.Millisecond)

	if len(mockObserver.events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(mockObserver.events))
	}

	if mockObserver.events[0] != event {
		t.Error("Received event does not match published event")
	}
}

// TestFileAuditObserver проверяет работу файлового наблюдателя
func TestFileAuditObserver(t *testing.T) {
	// Создаем временный файл
	tmpFile, err := os.CreateTemp("", "audit_test_*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	observer := NewFileAuditObserver(tmpFile.Name())
	event := NewAuditEvent([]string{"TestMetric1", "TestMetric2"}, "192.168.1.100")

	err = observer.Notify(event)
	if err != nil {
		t.Fatalf("Failed to notify file observer: %v", err)
	}

	// Читаем содержимое файла
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read audit file: %v", err)
	}

	content := string(data)

	// Проверяем наличие данных в файле
	if !strings.Contains(content, `"metrics":["TestMetric1","TestMetric2"]`) {
		t.Error("File does not contain correct metrics")
	}

	if !strings.Contains(content, `"ip_address":"192.168.1.100"`) {
		t.Error("File does not contain correct IP address")
	}

	// Проверяем, что строка заканчивается переносом строки
	if !strings.HasSuffix(content, "\n") {
		t.Error("File content should end with newline")
	}
}

// TestGetIPAddress проверяет извлечение IP-адреса из запроса
func TestGetIPAddress(t *testing.T) {
	tests := []struct {
		name           string
		remoteAddr     string
		xForwardedFor  string
		xRealIP        string
		expectedIP     string
	}{
		{
			name:       "Simple RemoteAddr",
			remoteAddr: "192.168.1.1:12345",
			expectedIP: "192.168.1.1",
		},
		{
			name:          "X-Forwarded-For single IP",
			remoteAddr:    "10.0.0.1:12345",
			xForwardedFor: "203.0.113.5",
			expectedIP:    "203.0.113.5",
		},
		{
			name:          "X-Forwarded-For multiple IPs",
			remoteAddr:    "10.0.0.1:12345",
			xForwardedFor: "203.0.113.5, 198.51.100.10, 192.0.2.1",
			expectedIP:    "203.0.113.5",
		},
		{
			name:       "X-Real-IP",
			remoteAddr: "10.0.0.1:12345",
			xRealIP:    "203.0.113.7",
			expectedIP: "203.0.113.7",
		},
		{
			name:          "X-Forwarded-For priority over X-Real-IP",
			remoteAddr:    "10.0.0.1:12345",
			xForwardedFor: "203.0.113.5",
			xRealIP:       "203.0.113.7",
			expectedIP:    "203.0.113.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем реальный HTTP-запрос с нужными заголовками
			req, err := createMockHTTPRequest(tt.remoteAddr, tt.xForwardedFor, tt.xRealIP)
			if err != nil {
				t.Fatalf("Failed to create mock request: %v", err)
			}

			ip := GetIPAddress(req)
			if ip != tt.expectedIP {
				t.Errorf("Expected IP %s, got %s", tt.expectedIP, ip)
			}
		})
	}
}

// Mock типы для тестирования

type mockObserver struct {
	events []*AuditEvent
}

func (m *mockObserver) Notify(event *AuditEvent) error {
	m.events = append(m.events, event)
	return nil
}

// createMockHTTPRequest создает реальный HTTP-запрос с заданными параметрами для тестирования
func createMockHTTPRequest(remoteAddr, xForwardedFor, xRealIP string) (*http.Request, error) {
	req, err := http.NewRequest("POST", "http://example.com", nil)
	if err != nil {
		return nil, err
	}

	req.RemoteAddr = remoteAddr

	if xForwardedFor != "" {
		req.Header.Set("X-Forwarded-For", xForwardedFor)
	}

	if xRealIP != "" {
		req.Header.Set("X-Real-IP", xRealIP)
	}

	return req, nil
}

