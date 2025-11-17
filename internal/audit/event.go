package audit

import (
	"encoding/json"
	"time"
)

// AuditEvent представляет событие аудита.
type AuditEvent struct {
	Timestamp int64    `json:"ts"`         // unix timestamp события
	Metrics   []string `json:"metrics"`    // наименование полученных метрик
	IPAddress string   `json:"ip_address"` // IP адрес входящего запроса
}

// NewAuditEvent создает новое событие аудита.
func NewAuditEvent(metrics []string, ipAddress string) *AuditEvent {
	return &AuditEvent{
		Timestamp: time.Now().Unix(),
		Metrics:   metrics,
		IPAddress: ipAddress,
	}
}

// ToJSON преобразует событие аудита в JSON.
func (e *AuditEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}
