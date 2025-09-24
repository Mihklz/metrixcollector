package agent

import (
	"math/rand"
	"runtime"
)

type MetricsSet struct {
	Gauges    map[string]float64
	PollCount int64
}

var pollCount int64

func Collect() MetricsSet {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	pollCount++

	return MetricsSet{
		PollCount: pollCount,
		Gauges: map[string]float64{
			"Alloc":         float64(m.Alloc),
			"BuckHashSys":   float64(m.BuckHashSys),
			"Frees":         float64(m.Frees),
			"GCCPUFraction": float64(m.GCCPUFraction),
			"GCSys":         float64(m.GCSys),
			"HeapAlloc":     float64(m.HeapAlloc),
			"HeapIdle":      float64(m.HeapIdle),
			"HeapInuse":     float64(m.HeapInuse),
			"HeapObjects":   float64(m.HeapObjects),
			"HeapReleased":  float64(m.HeapReleased),
			"HeapSys":       float64(m.HeapSys),
			"LastGC":        float64(m.LastGC),
			"Lookups":       float64(m.Lookups),
			"MCacheInuse":   float64(m.MCacheInuse),
			"MCacheSys":     float64(m.MCacheSys),
			"MSpanInuse":    float64(m.MSpanInuse),
			"MSpanSys":      float64(m.MSpanSys),
			"Mallocs":       float64(m.Mallocs),
			"NextGC":        float64(m.NextGC),
			"NumForcedGC":   float64(m.NumForcedGC),
			"NumGC":         float64(m.NumGC),
			"OtherSys":      float64(m.OtherSys),
			"PauseTotalNs":  float64(m.PauseTotalNs),
			"StackInuse":    float64(m.StackInuse),
			"StackSys":      float64(m.StackSys),
			"Sys":           float64(m.Sys),
			"TotalAlloc":    float64(m.TotalAlloc),
			"RandomValue":   rand.Float64(),
		},
	}
}

// CollectSystem собирает дополнительные системные метрики при помощи gopsutil.
// Возвращает карту gauge-метрик: TotalMemory, FreeMemory, CPUutilizationX
func CollectSystem() map[string]float64 {
	gauges := map[string]float64{}

	// Эти вызовы будут заполнены реализацией в файле collector_sys.go
	// для изоляции зависимости gopsutil на уровне сборки.
	total, free, cpu := readSystem()
	if total >= 0 {
		gauges["TotalMemory"] = total
	}
	if free >= 0 {
		gauges["FreeMemory"] = free
	}
	for i, v := range cpu {
		gauges["CPUutilization"+itoa(i+1)] = v
	}
	return gauges
}

// itoa — маленький helper без импорта strconv для минимального шума
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	digits := [20]byte{}
	pos := len(digits)
	for i > 0 {
		pos--
		digits[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(digits[pos:])
}

// readSystem реализуется в collector_sys.go
