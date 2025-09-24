package agent

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

// readSystem читает системные метрики при помощи gopsutil.
func readSystem() (total float64, free float64, cpuutil []float64) {
	// Память
	if vm, err := mem.VirtualMemoryWithContext(context.Background()); err == nil && vm != nil {
		total = float64(vm.Total)
		free = float64(vm.Free)
	} else {
		total = -1
		free = -1
	}

	// CPU utilization per CPU за небольшой интервал
	// Percent с interval>0 блокирует на интервал, используем неблокирующий способ WithContext + 0
	// затем fallback на 100ms для получения усреднения
	if vals, err := cpu.PercentWithContext(context.Background(), 0, true); err == nil {
		cpuutil = make([]float64, len(vals))
		for i, v := range vals {
			cpuutil[i] = v
		}
		return
	}

	if vals, err := cpu.PercentWithContext(context.Background(), 100*time.Millisecond, true); err == nil {
		cpuutil = make([]float64, len(vals))
		for i, v := range vals {
			cpuutil[i] = v
		}
	}
	return
}
