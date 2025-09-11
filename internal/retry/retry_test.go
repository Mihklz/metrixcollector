package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestExecute_Success(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts: 3,
		Delays:      []time.Duration{1 * time.Millisecond, 2 * time.Millisecond},
		Classifier:  NewDefaultErrorClassifier(),
	}

	callCount := 0
	operation := func() error {
		callCount++
		return nil
	}

	err := Execute(context.Background(), config, operation)
	
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount) // Должна быть только одна попытка
}

func TestExecute_RetriableError_SuccessAfterRetry(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts: 3,
		Delays:      []time.Duration{1 * time.Millisecond, 2 * time.Millisecond},
		Classifier:  NewDefaultErrorClassifier(),
	}

	callCount := 0
	operation := func() error {
		callCount++
		if callCount < 3 {
			return errors.New("connection refused") // Retriable error
		}
		return nil
	}

	err := Execute(context.Background(), config, operation)
	
	assert.NoError(t, err)
	assert.Equal(t, 3, callCount)
}

func TestExecute_NonRetriableError_NoRetry(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts: 3,
		Delays:      []time.Duration{1 * time.Millisecond, 2 * time.Millisecond},
		Classifier:  NewDefaultErrorClassifier(),
	}

	callCount := 0
	operation := func() error {
		callCount++
		return errors.New("invalid data") // Non-retriable error
	}

	err := Execute(context.Background(), config, operation)
	
	assert.Error(t, err)
	assert.Equal(t, 1, callCount) // Должна быть только одна попытка
}

func TestExecute_MaxAttemptsExceeded(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts: 3,
		Delays:      []time.Duration{1 * time.Millisecond, 2 * time.Millisecond},
		Classifier:  NewDefaultErrorClassifier(),
	}

	callCount := 0
	operation := func() error {
		callCount++
		return errors.New("connection refused") // Retriable error
	}

	err := Execute(context.Background(), config, operation)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operation failed after 3 attempts")
	assert.Equal(t, 3, callCount)
}

func TestExecute_ContextCancelled(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts: 5,
		Delays:      []time.Duration{10 * time.Millisecond, 20 * time.Millisecond},
		Classifier:  NewDefaultErrorClassifier(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	callCount := 0
	operation := func() error {
		callCount++
		if callCount == 2 {
			cancel() // Отменяем контекст после второй попытки
		}
		return errors.New("connection refused")
	}

	err := Execute(ctx, config, operation)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operation cancelled")
	assert.Equal(t, 2, callCount)
}

func TestExecuteWithTimeout(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts: 5,
		Delays:      []time.Duration{10 * time.Millisecond, 20 * time.Millisecond},
		Classifier:  NewDefaultErrorClassifier(),
	}

	callCount := 0
	operation := func() error {
		callCount++
		return errors.New("connection refused")
	}

	// Очень короткий таймаут
	err := ExecuteWithTimeout(context.Background(), config, 5*time.Millisecond, operation)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operation cancelled")
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()
	
	assert.Equal(t, 4, config.MaxAttempts)
	assert.Len(t, config.Delays, 3)
	assert.Equal(t, 1*time.Second, config.Delays[0])
	assert.Equal(t, 3*time.Second, config.Delays[1])
	assert.Equal(t, 5*time.Second, config.Delays[2])
	assert.NotNil(t, config.Classifier)
}

// MockErrorClassifier для тестирования
type MockErrorClassifier struct {
	mock.Mock
}

func (m *MockErrorClassifier) Classify(err error) ErrorClassification {
	args := m.Called(err)
	return args.Get(0).(ErrorClassification)
}
