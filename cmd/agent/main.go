package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Mihklz/metrixcollector/internal/agent"
	"github.com/Mihklz/metrixcollector/internal/config"
	"github.com/Mihklz/metrixcollector/internal/logger"
	"github.com/Mihklz/metrixcollector/internal/version"
)

func main() {
	version.Print()

	// Инициализируем логгер
	if err := logger.Initialize(); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Log.Sync()

	cfg := config.LoadAgentConfig()

	log.Println("Agent started")
	log.Printf("Poll interval: %v, Report interval: %v, Server: %s", cfg.PollInterval, cfg.ReportInterval, cfg.ServerAddr)

	// Создаем контекст с отменой для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Обработка сигналов для graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Создаем sender для отправки метрик
	sender := agent.NewMetricsSender(cfg.ServerAddr, cfg.Key)

	// Запускаем основной цикл в горутине
	var wg sync.WaitGroup
	wg.Add(1)
	go runAgent(ctx, &wg, cfg, sender)

	// Ждем сигнала завершения
	<-sigChan
	log.Println("Shutting down agent...")
	cancel()
	wg.Wait()
	log.Println("Agent stopped")
}

func runAgent(ctx context.Context, wg *sync.WaitGroup, cfg *config.AgentConfig, sender *agent.MetricsSender) {
	defer wg.Done()

	// Канал для передачи метрик от сборщика к пулу отправителей
	metricsCh := make(chan agent.MetricsSet, cfg.RateLimit*2)

	// Группа ожидания для воркеров
	var workersWg sync.WaitGroup

	// Запускаем пул воркеров для отправки (ограничение на одновременные исходящие запросы)
	for i := 0; i < cfg.RateLimit; i++ {
		workersWg.Add(1)
		go func() {
			defer workersWg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case ms, ok := <-metricsCh:
					if !ok {
						return
					}
					if err := sender.SendMetrics(ctx, ms); err != nil {
						log.Printf("failed to send metrics: %v", err)
					}
				}
			}
		}()
	}

	// Сборщик метрик: отдельно runtime и системные метрики
	tickerPoll := time.NewTicker(cfg.PollInterval)
	tickerReport := time.NewTicker(cfg.ReportInterval)
	defer tickerPoll.Stop()
	defer tickerReport.Stop()

	// Храним последнюю выборку метрик
	var current agent.MetricsSet

	// Стартуем дополнительный сборщик системных метрик в отдельной горутине
	// Он будет периодически обновлять current.Gauges дополнительными значениями
	sysDone := make(chan struct{})
	var sysMu sync.Mutex
	latestSys := map[string]float64{}
	go func() {
		defer close(sysDone)
		sysTicker := time.NewTicker(cfg.PollInterval)
		defer sysTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-sysTicker.C:
				sys := agent.CollectSystem()
				sysMu.Lock()
				latestSys = sys
				sysMu.Unlock()
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			// Завершаем: закрываем канал, ждём воркеров
			close(metricsCh)
			workersWg.Wait()
			return
		case <-tickerPoll.C:
			// Сбор runtime метрик
			current = agent.Collect()
			// Добавим системные метрики
			sysMu.Lock()
			for k, v := range latestSys {
				if current.Gauges == nil {
					current.Gauges = map[string]float64{}
				}
				current.Gauges[k] = v
			}
			sysMu.Unlock()
		case <-tickerReport.C:
			// Отправляем снимок через канал в пул воркеров
			snapshot := agent.MetricsSet{
				Gauges:    make(map[string]float64, len(current.Gauges)),
				PollCount: current.PollCount,
			}
			for k, v := range current.Gauges {
				snapshot.Gauges[k] = v
			}
			select {
			case metricsCh <- snapshot:
			case <-ctx.Done():
				close(metricsCh)
				workersWg.Wait()
				return
			}
		}
	}
}
