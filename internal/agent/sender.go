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

	"github.com/Mihklz/metrixcollector/internal/crypto"
	"github.com/Mihklz/metrixcollector/internal/logger"
	models "github.com/Mihklz/metrixcollector/internal/model"
	"github.com/Mihklz/metrixcollector/internal/retry"
)

type MetricsSender struct {
	client      *http.Client
	serverAddr  string
	retryConfig *retry.RetryConfig
	key         string // ключ для подписи данных
}

func NewMetricsSender(serverAddr string, key string) *MetricsSender {
	return &MetricsSender{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		serverAddr:  serverAddr,
		retryConfig: retry.DefaultRetryConfig(),
		key:         key,
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
	// Пытаемся отправить всё одним batch запросом с retry-логикой
	err := retry.Execute(ctx, s.retryConfig, func() error {
		return s.SendMetricsBatch(ctx, metrics)
	})

	if err != nil {
		logger.Log.Warn("Batch send failed after retries, falling back to individual requests", zap.Error(err))

		// Fallback на отдельные запросы для обратной совместимости
		return s.sendMetricsIndividually(ctx, metrics)
	}

	return nil
}

// SendMetricsBatch отправляет все метрики одним batch запросом
func (s *MetricsSender) SendMetricsBatch(ctx context.Context, metrics MetricsSet) error {
	if len(metrics.Gauges) == 0 && metrics.PollCount == 0 {
		return nil // Не отправляем пустые батчи
	}

	// Собираем все метрики в один слайс
	var allMetrics []models.Metrics

	// Добавляем gauge метрики
	for name, value := range metrics.Gauges {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		v := value // создаём копию для указателя
		allMetrics = append(allMetrics, models.Metrics{
			ID:    name,
			MType: models.Gauge,
			Value: &v,
		})
	}

	// Добавляем counter метрику
	if metrics.PollCount > 0 {
		delta := metrics.PollCount
		allMetrics = append(allMetrics, models.Metrics{
			ID:    "PollCount",
			MType: models.Counter,
			Delta: &delta,
		})
	}

	// Сериализуем в JSON
	jsonData, err := json.Marshal(allMetrics)
	if err != nil {
		return fmt.Errorf("marshal batch metrics error: %w", err)
	}

	// Сжимаем данные в gzip
	compressedData, err := compressData(jsonData)
	if err != nil {
		return fmt.Errorf("compress batch data error: %w", err)
	}

	// Создаём POST запрос к /updates/
	url := fmt.Sprintf("%s/updates/", s.serverAddr)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(compressedData))
	if err != nil {
		return fmt.Errorf("create batch request error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	// Добавляем хеш в заголовок, если есть ключ
	if s.key != "" {
		hash := crypto.CalculateHMAC(jsonData, s.key)
		req.Header.Set("HashSHA256", hash)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send batch error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	logger.Log.Info("Batch metrics sent successfully", zap.Int("count", len(allMetrics)))
	return nil
}

// sendMetricsIndividually отправляет метрики по одной (fallback)
func (s *MetricsSender) sendMetricsIndividually(ctx context.Context, metrics MetricsSet) error {
	// Отправляем gauge метрики с retry-логикой
	for name, value := range metrics.Gauges {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := retry.Execute(ctx, s.retryConfig, func() error {
			return s.sendGauge(name, value)
		})

		if err != nil {
			logger.Log.Error("Failed to send gauge metric after retries",
				zap.String("name", name),
				zap.Float64("value", value),
				zap.Error(err),
			)
			// Продолжаем отправку остальных метрик
		}
	}

	// Отправляем counter метрику с retry-логикой
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	err := retry.Execute(ctx, s.retryConfig, func() error {
		return s.sendCounter("PollCount", metrics.PollCount)
	})

	if err != nil {
		logger.Log.Error("Failed to send counter metric after retries",
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

	// Добавляем хеш в заголовок, если есть ключ
	if s.key != "" {
		hash := crypto.CalculateHMAC(jsonData, s.key)
		req.Header.Set("HashSHA256", hash)
	}

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

	// Добавляем хеш в заголовок, если есть ключ
	if s.key != "" {
		hash := crypto.CalculateHMAC(jsonData, s.key)
		req.Header.Set("HashSHA256", hash)
	}

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
