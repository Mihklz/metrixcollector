package service

import (
	"fmt"
	"testing"

	models "github.com/Mihklz/metrixcollector/internal/model"
	"github.com/Mihklz/metrixcollector/internal/repository"
)

const benchmarkBatchSize = 256

func prepareBenchmarkMetrics() []models.Metrics {
	metrics := make([]models.Metrics, 0, benchmarkBatchSize)
	for i := 0; i < benchmarkBatchSize/2; i++ {
		v := float64(i)
		metrics = append(metrics, models.Metrics{
			ID:    fmt.Sprintf("gauge_%d", i),
			MType: models.Gauge,
			Value: &v,
		})
	}
	for i := 0; i < benchmarkBatchSize/2; i++ {
		d := int64(i)
		metrics = append(metrics, models.Metrics{
			ID:    fmt.Sprintf("counter_%d", i),
			MType: models.Counter,
			Delta: &d,
		})
	}
	return metrics
}

func BenchmarkMetricsServiceUpdateBatch(b *testing.B) {
	storage := repository.NewMemStorage()
	svc := NewMetricsService(storage)
	metrics := prepareBenchmarkMetrics()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := svc.UpdateBatch(metrics); err != nil {
			b.Fatalf("update batch failed: %v", err)
		}
	}
}

