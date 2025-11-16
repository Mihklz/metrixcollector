package audit

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestAuditEventEmptyMetrics проверяет создание события с пустым списком метрик
func TestAuditEventEmptyMetrics(t *testing.T) {
	event := NewAuditEvent([]string{}, "192.168.1.1")

	if event == nil {
		t.Fatal("NewAuditEvent returned nil")
	}

	if len(event.Metrics) != 0 {
		t.Errorf("Expected empty metrics slice, got %d metrics", len(event.Metrics))
	}

	if event.IPAddress != "192.168.1.1" {
		t.Errorf("Expected IP 192.168.1.1, got %s", event.IPAddress)
	}
}

// TestAuditEventNilMetrics проверяет создание события с nil метриками
func TestAuditEventNilMetrics(t *testing.T) {
	event := NewAuditEvent(nil, "192.168.1.1")

	if event == nil {
		t.Fatal("NewAuditEvent returned nil")
	}

	// nil slice должен сериализоваться в JSON как null
	jsonData, err := event.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	if !strings.Contains(string(jsonData), `"metrics":null`) {
		t.Error("Nil metrics should serialize as null")
	}
}

// TestAuditEventEmptyIPAddress проверяет создание события с пустым IP
func TestAuditEventEmptyIPAddress(t *testing.T) {
	event := NewAuditEvent([]string{"Metric1"}, "")

	if event.IPAddress != "" {
		t.Errorf("Expected empty IP address, got %s", event.IPAddress)
	}

	// Проверяем, что событие всё равно сериализуется
	jsonData, err := event.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	if !strings.Contains(string(jsonData), `"ip_address":""`) {
		t.Error("Empty IP address should be serialized")
	}
}

// TestAuditEventLargeMetricsList проверяет работу с большим количеством метрик
func TestAuditEventLargeMetricsList(t *testing.T) {
	// Создаем большой список метрик
	metrics := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		metrics[i] = "Metric" + string(rune(i))
	}

	event := NewAuditEvent(metrics, "192.168.1.1")

	if len(event.Metrics) != 1000 {
		t.Errorf("Expected 1000 metrics, got %d", len(event.Metrics))
	}

	// Проверяем, что событие сериализуется без ошибок
	jsonData, err := event.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert large event to JSON: %v", err)
	}

	// Проверяем, что можем десериализовать обратно
	var decoded AuditEvent
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal large event: %v", err)
	}

	if len(decoded.Metrics) != 1000 {
		t.Errorf("After unmarshal: expected 1000 metrics, got %d", len(decoded.Metrics))
	}
}

// TestAuditEventSpecialCharacters проверяет обработку специальных символов
func TestAuditEventSpecialCharacters(t *testing.T) {
	metrics := []string{
		"Metric with spaces",
		"Metric-with-dashes",
		"Metric_with_underscores",
		"Metric.with.dots",
		"Метрика на русском",
		"メトリック",
	}
	ipAddress := "::1" // IPv6

	event := NewAuditEvent(metrics, ipAddress)

	jsonData, err := event.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert event with special chars to JSON: %v", err)
	}

	// Проверяем, что можем десериализовать
	var decoded AuditEvent
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal event with special chars: %v", err)
	}

	if len(decoded.Metrics) != len(metrics) {
		t.Errorf("Expected %d metrics, got %d", len(metrics), len(decoded.Metrics))
	}

	for i, metric := range metrics {
		if decoded.Metrics[i] != metric {
			t.Errorf("Metric %d: expected %s, got %s", i, metric, decoded.Metrics[i])
		}
	}

	if decoded.IPAddress != ipAddress {
		t.Errorf("Expected IP %s, got %s", ipAddress, decoded.IPAddress)
	}
}

// TestAuditEventTimestampAccuracy проверяет точность timestamp
func TestAuditEventTimestampAccuracy(t *testing.T) {
	beforeTime := time.Now().Unix()
	event := NewAuditEvent([]string{"Metric1"}, "127.0.0.1")
	afterTime := time.Now().Unix()

	if event.Timestamp < beforeTime {
		t.Errorf("Timestamp %d is before creation time %d", event.Timestamp, beforeTime)
	}

	if event.Timestamp > afterTime {
		t.Errorf("Timestamp %d is after creation time %d", event.Timestamp, afterTime)
	}
}

// TestAuditEventJSONRoundtrip проверяет полный цикл сериализации/десериализации
func TestAuditEventJSONRoundtrip(t *testing.T) {
	original := &AuditEvent{
		Timestamp: 1234567890,
		Metrics:   []string{"Metric1", "Metric2", "Metric3"},
		IPAddress: "192.168.1.100",
	}

	// Сериализуем
	jsonData, err := original.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	// Десериализуем
	var decoded AuditEvent
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to deserialize: %v", err)
	}

	// Сравниваем
	if decoded.Timestamp != original.Timestamp {
		t.Errorf("Timestamp mismatch: expected %d, got %d", original.Timestamp, decoded.Timestamp)
	}

	if decoded.IPAddress != original.IPAddress {
		t.Errorf("IP address mismatch: expected %s, got %s", original.IPAddress, decoded.IPAddress)
	}

	if len(decoded.Metrics) != len(original.Metrics) {
		t.Errorf("Metrics count mismatch: expected %d, got %d", len(original.Metrics), len(decoded.Metrics))
	}

	for i, metric := range original.Metrics {
		if decoded.Metrics[i] != metric {
			t.Errorf("Metric %d mismatch: expected %s, got %s", i, metric, decoded.Metrics[i])
		}
	}
}

// TestAuditEventJSONFormat проверяет формат JSON
func TestAuditEventJSONFormat(t *testing.T) {
	event := &AuditEvent{
		Timestamp: 1700000000,
		Metrics:   []string{"Alloc", "Frees"},
		IPAddress: "192.168.0.42",
	}

	jsonData, err := event.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	jsonStr := string(jsonData)

	// Проверяем ключи и формат
	expectedParts := []string{
		`"ts":1700000000`,
		`"metrics":["Alloc","Frees"]`,
		`"ip_address":"192.168.0.42"`,
	}

	for _, part := range expectedParts {
		if !strings.Contains(jsonStr, part) {
			t.Errorf("JSON does not contain expected part: %s\nGot: %s", part, jsonStr)
		}
	}
}

