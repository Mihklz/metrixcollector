package agent

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

type MetricsSender struct {
	client     *http.Client
	serverAddr string
}

func NewMetricsSender(serverAddr string) *MetricsSender {
	return &MetricsSender{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		serverAddr: serverAddr,
	}
}

func (s *MetricsSender) SendMetrics(ctx context.Context, metrics MetricsSet) error {
	// Отправляем gauge метрики
	for name, value := range metrics.Gauges {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := s.sendGauge(name, value); err != nil {
			log.Printf("failed to send gauge %s: %v", name, err)
			// Продолжаем отправку остальных метрик
		}
	}

	// Отправляем counter метрику
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := s.sendCounter("PollCount", metrics.PollCount); err != nil {
		log.Printf("failed to send counter PollCount: %v", err)
		return err
	}

	return nil
}

func (s *MetricsSender) sendGauge(name string, value float64) error {
	url := fmt.Sprintf("%s/update/gauge/%s/%f", s.serverAddr, name, value)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("create gauge request error: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send gauge error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (s *MetricsSender) sendCounter(name string, value int64) error {
	url := fmt.Sprintf("%s/update/counter/%s/%d", s.serverAddr, name, value)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("create counter request error: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send counter error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
} 