package retry

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/Mihklz/metrixcollector/internal/logger"
)

// RetryConfig конфигурация для retry-логики
type RetryConfig struct {
	MaxAttempts int           // Максимальное количество попыток (включая первую)
	Delays      []time.Duration // Задержки между попытками
	Classifier  ErrorClassifier // Классификатор ошибок
}

// DefaultRetryConfig возвращает конфигурацию по умолчанию
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts: 4, // 1 основная + 3 повтора
		Delays:      []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second},
		Classifier:  NewDefaultErrorClassifier(),
	}
}

// Execute выполняет функцию с retry-логикой
func Execute(ctx context.Context, config *RetryConfig, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		// Проверяем контекст перед каждой попыткой
		select {
		case <-ctx.Done():
			return fmt.Errorf("operation cancelled: %w", ctx.Err())
		default:
		}

		// Выполняем операцию
		err := operation()
		if err == nil {
		// Успешно выполнено
		if attempt > 0 && logger.Log != nil {
			logger.Log.Info("Operation succeeded after retry",
				zap.Int("attempt", attempt+1),
				zap.Int("max_attempts", config.MaxAttempts),
			)
		}
			return nil
		}

		lastErr = err

		// Если это последняя попытка, не ждем
		if attempt == config.MaxAttempts-1 {
			break
		}

		// Классифицируем ошибку
		classification := config.Classifier.Classify(err)
		if classification == NonRetriable {
			if logger.Log != nil {
				logger.Log.Warn("Non-retriable error encountered, stopping retries",
					zap.Error(err),
					zap.Int("attempt", attempt+1),
				)
			}
			return err
		}

		// Логируем retriable ошибку
		if logger.Log != nil {
			logger.Log.Warn("Retriable error encountered, will retry",
				zap.Error(err),
				zap.Int("attempt", attempt+1),
				zap.Int("max_attempts", config.MaxAttempts),
			)
		}

		// Ждем перед следующей попыткой
		delay := config.Delays[attempt]
		if logger.Log != nil {
			logger.Log.Debug("Waiting before retry",
				zap.Duration("delay", delay),
				zap.Int("next_attempt", attempt+2),
			)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("operation cancelled during retry delay: %w", ctx.Err())
		case <-time.After(delay):
			// Продолжаем к следующей попытке
		}
	}

	// Все попытки исчерпаны
	return fmt.Errorf("operation failed after %d attempts: %w", config.MaxAttempts, lastErr)
}

// ExecuteWithTimeout выполняет операцию с retry-логикой и общим таймаутом
func ExecuteWithTimeout(ctx context.Context, config *RetryConfig, timeout time.Duration, operation func() error) error {
	// Создаем контекст с таймаутом
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return Execute(timeoutCtx, config, operation)
}
