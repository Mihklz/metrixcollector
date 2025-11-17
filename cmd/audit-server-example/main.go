package main

// Это пример простого HTTP-сервера для приема событий аудита.
// Используйте его для тестирования HTTP-наблюдателя аудита.
//
// Запуск:
//   go run internal/audit/example_audit_server.go
//
// Затем запустите основной сервер с параметром:
//   ./server --audit-url=http://localhost:9090/audit

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// AuditEvent представляет событие аудита
type AuditEvent struct {
	Timestamp int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

func handleAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method allowed", http.StatusMethodNotAllowed)
		return
	}

	// Читаем тело запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v\n", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Декодируем JSON
	var event AuditEvent
	if err := json.Unmarshal(body, &event); err != nil {
		log.Printf("Error decoding JSON: %v\n", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Выводим информацию о событии
	timestamp := time.Unix(event.Timestamp, 0).Format("2006-01-02 15:04:05")
	fmt.Printf("[%s] Received audit event from %s: metrics=%v\n",
		timestamp, event.IPAddress, event.Metrics)

	// Отправляем успешный ответ
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	http.HandleFunc("/audit", handleAudit)

	addr := ":9090"
	fmt.Printf("Starting audit receiver server on %s\n", addr)
	fmt.Println("Waiting for audit events at http://localhost:9090/audit")
	fmt.Println("Press Ctrl+C to stop")

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed to start: %v\n", err)
	}
}
