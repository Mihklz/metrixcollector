package retry

import (
	"errors"
	"net"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

func TestDefaultErrorClassifier_Classify(t *testing.T) {
	classifier := NewDefaultErrorClassifier()

	tests := []struct {
		name           string
		err            error
		expectedResult ErrorClassification
	}{
		{
			name:           "nil error",
			err:            nil,
			expectedResult: NonRetriable,
		},
		{
			name:           "network timeout error",
			err:            &net.OpError{Op: "read", Net: "tcp", Err: &timeoutError{}},
			expectedResult: Retriable,
		},
		{
			name:           "connection refused error",
			err:            errors.New("dial tcp 127.0.0.1:8080: connect: connection refused"),
			expectedResult: Retriable,
		},
		{
			name:           "postgres connection exception",
			err:            &pgconn.PgError{Code: pgerrcode.ConnectionException},
			expectedResult: Retriable,
		},
		{
			name:           "postgres serialization failure",
			err:            &pgconn.PgError{Code: pgerrcode.SerializationFailure},
			expectedResult: Retriable,
		},
		{
			name:           "postgres unique violation",
			err:            &pgconn.PgError{Code: pgerrcode.UniqueViolation},
			expectedResult: NonRetriable,
		},
		{
			name:           "http 502 bad gateway",
			err:            errors.New("502 Bad Gateway"),
			expectedResult: Retriable,
		},
		{
			name:           "http 503 service unavailable",
			err:            errors.New("503 Service Unavailable"),
			expectedResult: Retriable,
		},
		{
			name:           "validation error",
			err:            errors.New("invalid input data"),
			expectedResult: NonRetriable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.Classify(tt.err)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "timeout error",
			err:      &timeoutError{},
			expected: true,
		},
		{
			name:     "connection refused",
			err:      errors.New("dial tcp: connect: connection refused"),
			expected: true,
		},
		{
			name:     "connection timeout",
			err:      errors.New("connection timeout"),
			expected: true,
		},
		{
			name:     "i/o timeout",
			err:      errors.New("i/o timeout"),
			expected: true,
		},
		{
			name:     "validation error",
			err:      errors.New("invalid data"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNetworkError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsPostgresRetriableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "connection exception",
			err:      &pgconn.PgError{Code: pgerrcode.ConnectionException},
			expected: true,
		},
		{
			name:     "serialization failure",
			err:      &pgconn.PgError{Code: pgerrcode.SerializationFailure},
			expected: true,
		},
		{
			name:     "unique violation",
			err:      &pgconn.PgError{Code: pgerrcode.UniqueViolation},
			expected: false,
		},
		{
			name:     "syntax error",
			err:      &pgconn.PgError{Code: pgerrcode.SyntaxError},
			expected: false,
		},
		{
			name:     "non-postgres error",
			err:      errors.New("some error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPostgresRetriableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// timeoutError реализует net.Error для тестов
type timeoutError struct{}

func (e *timeoutError) Error() string   { return "timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }
