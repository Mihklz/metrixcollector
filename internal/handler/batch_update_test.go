package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/Mihklz/metrixcollector/internal/logger"
	models "github.com/Mihklz/metrixcollector/internal/model"
	"github.com/Mihklz/metrixcollector/internal/repository"
	"github.com/Mihklz/metrixcollector/internal/service"
)

func init() {
	// Инициализируем логгер для тестов
	logger.Log = zap.NewNop() // Используем no-op логгер для тестов
}

func TestBatchUpdateHandler(t *testing.T) {
	storage := repository.NewMemStorage()
	metricsService := service.NewMetricsService(storage)
	handler := NewBatchUpdateHandler(metricsService, "")

	tests := []struct {
		name           string
		metrics        []models.Metrics
		expectedStatus int
	}{
		{
			name: "successful batch update",
			metrics: []models.Metrics{
				{
					ID:    "test_gauge",
					MType: models.Gauge,
					Value: func() *float64 { v := 123.45; return &v }(),
				},
				{
					ID:    "test_counter",
					MType: models.Counter,
					Delta: func() *int64 { v := int64(10); return &v }(),
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "empty batch",
			metrics:        []models.Metrics{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing gauge value",
			metrics: []models.Metrics{
				{
					ID:    "test_gauge",
					MType: models.Gauge,
					// Value отсутствует
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing counter delta",
			metrics: []models.Metrics{
				{
					ID:    "test_counter",
					MType: models.Counter,
					// Delta отсутствует
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "unknown metric type",
			metrics: []models.Metrics{
				{
					ID:    "test_unknown",
					MType: "unknown",
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Сериализуем метрики в JSON
			jsonData, err := json.Marshal(tt.metrics)
			require.NoError(t, err)

			// Создаём HTTP запрос
			req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(context.Background())
			w := httptest.NewRecorder()

			// Вызываем handler
			handler(w, req)

			// Проверяем статус код
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Если операция успешна, проверяем, что метрики сохранились
			if tt.expectedStatus == http.StatusOK {
				for _, metric := range tt.metrics {
					switch metric.MType {
					case models.Gauge:
						if metric.Value != nil {
							value, exists := storage.GetGauge(metric.ID)
							assert.True(t, exists)
							assert.Equal(t, *metric.Value, float64(value))
						}
					case models.Counter:
						if metric.Delta != nil {
							value, exists := storage.GetCounter(metric.ID)
							assert.True(t, exists)
							assert.Equal(t, *metric.Delta, int64(value))
						}
					}
				}
			}
		})
	}
}

func TestBatchUpdateHandlerMethods(t *testing.T) {
	storage := repository.NewMemStorage()
	metricsService := service.NewMetricsService(storage)
	handler := NewBatchUpdateHandler(metricsService, "")

	tests := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{
			name:           "POST method allowed",
			method:         http.MethodPost,
			expectedStatus: http.StatusBadRequest, // Пустое тело
		},
		{
			name:           "GET method not allowed",
			method:         http.MethodGet,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "PUT method not allowed",
			method:         http.MethodPut,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/updates/", nil)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestBatchUpdateHandlerContentType(t *testing.T) {
	storage := repository.NewMemStorage()
	metricsService := service.NewMetricsService(storage)
	handler := NewBatchUpdateHandler(metricsService, "")

	tests := []struct {
		name           string
		contentType    string
		expectedStatus int
	}{
		{
			name:           "correct content type",
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest, // Пустое тело
		},
		{
			name:           "incorrect content type",
			contentType:    "text/plain",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing content type",
			contentType:    "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/updates/", nil)
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			w := httptest.NewRecorder()

			handler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
