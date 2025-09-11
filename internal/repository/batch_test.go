package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	models "github.com/Mihklz/metrixcollector/internal/model"
)

func TestMemStorage_UpdateBatch(t *testing.T) {
	storage := NewMemStorage()

	tests := []struct {
		name        string
		metrics     []models.Metrics
		expectError bool
		errorMsg    string
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
			expectError: false,
		},
		{
			name:        "empty batch",
			metrics:     []models.Metrics{},
			expectError: false,
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
			expectError: true,
			errorMsg:    "gauge metric test_gauge missing value",
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
			expectError: true,
			errorMsg:    "counter metric test_counter missing delta",
		},
		{
			name: "unknown metric type",
			metrics: []models.Metrics{
				{
					ID:    "test_unknown",
					MType: "unknown",
				},
			},
			expectError: true,
			errorMsg:    "unsupported metric type: unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := storage.UpdateBatch(tt.metrics)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)

				// Проверяем, что метрики сохранились
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

func TestMemStorage_ImplementsBatchStorage(t *testing.T) {
	storage := NewMemStorage()

	// Проверяем, что MemStorage реализует BatchStorage интерфейс
	var _ BatchStorage = storage

	// Также проверяем через интерфейс
	var batchStorage BatchStorage = storage
	assert.NotNil(t, batchStorage)
}
