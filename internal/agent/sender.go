package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/Mihklz/metrixcollector/internal/logger"
	models "github.com/Mihklz/metrixcollector/internal/model"
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

// compressData сжимает данные в формате gzip
func compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)

	_, err := gz.Write(data)
	if err != nil {
		return nil, err
	}

	err = gz.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
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
			logger.Log.Error("Failed to send gauge metric",
				zap.String("name", name),
				zap.Float64("value", value),
				zap.Error(err),
			)
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
		logger.Log.Error("Failed to send counter metric",
			zap.String("name", "PollCount"),
			zap.Int64("value", metrics.PollCount),
			zap.Error(err),
		)
		return err
	}

	return nil
}

func (s *MetricsSender) sendGauge(name string, value float64) error {
	// Создаём структуру для JSON API
	metric := models.Metrics{
		ID:    name,
		MType: models.Gauge,
		Value: &value,
	}

	// Сериализуем в JSON
	jsonData, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("marshal gauge metric error: %w", err)
	}

	// Сжимаем данные в gzip
	compressedData, err := compressData(jsonData)
	if err != nil {
		return fmt.Errorf("compress gauge data error: %w", err)
	}

	// Создаём POST запрос к /update
	url := fmt.Sprintf("%s/update", s.serverAddr)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(compressedData))
	if err != nil {
		return fmt.Errorf("create gauge request error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

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
	// Создаём структуру для JSON API
	metric := models.Metrics{
		ID:    name,
		MType: models.Counter,
		Delta: &value,
	}

	// Сериализуем в JSON
	jsonData, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("marshal counter metric error: %w", err)
	}

	// Сжимаем данные в gzip
	compressedData, err := compressData(jsonData)
	if err != nil {
		return fmt.Errorf("compress counter data error: %w", err)
	}

	// Создаём POST запрос к /update
	url := fmt.Sprintf("%s/update", s.serverAddr)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(compressedData))
	if err != nil {
		return fmt.Errorf("create counter request error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

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
