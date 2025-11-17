package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// TestFileAuditObserverMultipleEvents проверяет запись нескольких событий
func TestFileAuditObserverMultipleEvents(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "audit_multiple_*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	observer := NewFileAuditObserver(tmpFile.Name())

	// Создаем и отправляем несколько событий
	events := []*AuditEvent{
		NewAuditEvent([]string{"Metric1"}, "192.168.1.1"),
		NewAuditEvent([]string{"Metric2", "Metric3"}, "192.168.1.2"),
		NewAuditEvent([]string{"Metric4"}, "192.168.1.3"),
	}

	for _, event := range events {
		if err := observer.Notify(event); err != nil {
			t.Fatalf("Failed to notify: %v", err)
		}
	}

	// Читаем файл и проверяем
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}

	// Проверяем, что каждая строка - валидный JSON
	for i, line := range lines {
		var event AuditEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Errorf("Line %d is not valid JSON: %v", i, err)
		}
	}
}

// TestFileAuditObserverInvalidPath проверяет обработку невалидного пути
func TestFileAuditObserverInvalidPath(t *testing.T) {
	// Используем путь к несуществующей директории
	invalidPath := "/nonexistent/directory/that/does/not/exist/audit.log"
	observer := NewFileAuditObserver(invalidPath)

	event := NewAuditEvent([]string{"TestMetric"}, "127.0.0.1")
	err := observer.Notify(event)

	if err == nil {
		t.Error("Expected error for invalid path, got nil")
	}
}

// TestFileAuditObserverConcurrentWrites проверяет конкурентную запись
func TestFileAuditObserverConcurrentWrites(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "audit_concurrent_*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	observer := NewFileAuditObserver(tmpFile.Name())

	// Конкурентно записываем события
	numEvents := 50
	var wg sync.WaitGroup
	wg.Add(numEvents)

	for i := 0; i < numEvents; i++ {
		go func(idx int) {
			defer wg.Done()
			event := NewAuditEvent([]string{"Metric" + string(rune(idx))}, "127.0.0.1")
			observer.Notify(event)
		}(i)
	}

	wg.Wait()

	// Читаем файл и проверяем количество строк
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != numEvents {
		t.Errorf("Expected %d lines, got %d", numEvents, len(lines))
	}

	// Проверяем, что все строки - валидный JSON
	for i, line := range lines {
		var event AuditEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Errorf("Line %d is not valid JSON: %v", i, err)
		}
	}
}

// TestFileAuditObserverCreatesFile проверяет создание файла, если его нет
func TestFileAuditObserverCreatesFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "new_audit.log")

	// Убеждаемся, что файл не существует
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatal("File should not exist initially")
	}

	observer := NewFileAuditObserver(filePath)
	event := NewAuditEvent([]string{"TestMetric"}, "127.0.0.1")

	if err := observer.Notify(event); err != nil {
		t.Fatalf("Failed to notify: %v", err)
	}

	// Проверяем, что файл был создан
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("File should have been created")
	}

	// Проверяем содержимое
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if len(data) == 0 {
		t.Error("File should not be empty")
	}
}

// TestFileAuditObserverAppends проверяет, что данные добавляются, а не перезаписываются
func TestFileAuditObserverAppends(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "audit_append_*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Записываем начальные данные
	initialData := `{"ts":1000000,"metrics":["Initial"],"ip_address":"1.1.1.1"}` + "\n"
	if _, err := tmpFile.WriteString(initialData); err != nil {
		t.Fatalf("Failed to write initial data: %v", err)
	}
	tmpFile.Close()

	// Создаем наблюдателя и добавляем новое событие
	observer := NewFileAuditObserver(tmpFile.Name())
	event := NewAuditEvent([]string{"NewMetric"}, "2.2.2.2")

	if err := observer.Notify(event); err != nil {
		t.Fatalf("Failed to notify: %v", err)
	}

	// Читаем файл
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines (initial + new), got %d", len(lines))
	}

	// Проверяем, что начальные данные сохранились
	if !strings.Contains(lines[0], "Initial") {
		t.Error("Initial data was overwritten instead of appended to")
	}

	// Проверяем новые данные
	if !strings.Contains(lines[1], "NewMetric") {
		t.Error("New data was not appended correctly")
	}
}

