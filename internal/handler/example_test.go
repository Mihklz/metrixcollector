package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"

	models "github.com/Mihklz/metrixcollector/internal/model"
	"github.com/Mihklz/metrixcollector/internal/repository"
)

// ExampleNewJSONUpdateHandler демонстрирует работу JSON-эндпоинта обновления метрик.
func ExampleNewJSONUpdateHandler() {
	storage := repository.NewMemStorage()
	handler := NewJSONUpdateHandler(storage, "", nil)

	body, _ := json.Marshal(models.Metrics{
		ID:    "Alloc",
		MType: models.Gauge,
		Value: func(v float64) *float64 { return &v }(42.5),
	})

	req := httptest.NewRequest("POST", "/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler(rec, req)

	value, _ := storage.GetGauge("Alloc")
	fmt.Printf("Alloc gauge: %.1f\n", value)

	// Output:
	// Alloc gauge: 42.5
}
