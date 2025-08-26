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
)

func main() {
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
	sender := agent.NewMetricsSender(cfg.ServerAddr)

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

	tickerPoll := time.NewTicker(cfg.PollInterval)
	tickerReport := time.NewTicker(cfg.ReportInterval)
	defer tickerPoll.Stop()
	defer tickerReport.Stop()

	var currentMetrics agent.MetricsSet
	var reportWg sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			// Ждем завершения всех отправок метрик
			reportWg.Wait()
			return
		case <-tickerPoll.C:
			currentMetrics = agent.Collect()
		case <-tickerReport.C:
			// Запускаем отправку метрик в отдельной горутине с контролем
			reportWg.Add(1)
			go func(metrics agent.MetricsSet) {
				defer reportWg.Done()
				if err := sender.SendMetrics(ctx, metrics); err != nil {
					log.Printf("failed to send metrics: %v", err)
				}
			}(currentMetrics)
		}
	}
}
