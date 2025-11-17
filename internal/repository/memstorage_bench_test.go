package repository

import (
	"fmt"
	"path/filepath"
	"testing"

	models "github.com/Mihklz/metrixcollector/internal/model"
)

const benchmarkMetricPool = 1024

func BenchmarkMemStorageUpdateGauge(b *testing.B) {
	storage := NewMemStorage()
	value := "123.456"

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("gauge_%d", i%benchmarkMetricPool)
		if err := storage.Update(models.Gauge, name, value); err != nil {
			b.Fatalf("update gauge failed: %v", err)
		}
	}
}

func BenchmarkMemStorageUpdateCounter(b *testing.B) {
	storage := NewMemStorage()
	value := "10"

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("counter_%d", i%benchmarkMetricPool)
		if err := storage.Update(models.Counter, name, value); err != nil {
			b.Fatalf("update counter failed: %v", err)
		}
	}
}

func BenchmarkMemStorageSaveToFile(b *testing.B) {
	storage := NewMemStorage()
	for i := 0; i < 5000; i++ {
		if err := storage.Update(models.Gauge, fmt.Sprintf("gauge_%d", i), "42.42"); err != nil {
			b.Fatalf("prepare gauge failed: %v", err)
		}
		if err := storage.Update(models.Counter, fmt.Sprintf("counter_%d", i), "5"); err != nil {
			b.Fatalf("prepare counter failed: %v", err)
		}
	}

	dir := b.TempDir()
	filePath := filepath.Join(dir, "metrics.json")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := storage.SaveToFile(filePath); err != nil {
			b.Fatalf("save to file failed: %v", err)
		}
	}
}
