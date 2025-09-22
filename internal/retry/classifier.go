package retry

import (
	"errors"
	"net"
	"strings"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrorClassification тип для классификации ошибок
type ErrorClassification int

const (
	// NonRetriable - операцию не следует повторять
	NonRetriable ErrorClassification = iota

	// Retriable - операцию можно повторить
	Retriable
)

// ErrorClassifier интерфейс для классификации ошибок
type ErrorClassifier interface {
	Classify(err error) ErrorClassification
}

// DefaultErrorClassifier классификатор ошибок по умолчанию
type DefaultErrorClassifier struct{}

// NewDefaultErrorClassifier создает новый классификатор ошибок
func NewDefaultErrorClassifier() *DefaultErrorClassifier {
	return &DefaultErrorClassifier{}
}

// Classify классифицирует ошибку и возвращает ErrorClassification
func (c *DefaultErrorClassifier) Classify(err error) ErrorClassification {
	if err == nil {
		return NonRetriable
	}

	// Проверяем сетевые ошибки (retriable)
	if isNetworkError(err) {
		return Retriable
	}

	// Проверяем ошибки PostgreSQL
	if isPostgresRetriableError(err) {
		return Retriable
	}

	// Проверяем временные ошибки HTTP
	if isTemporaryHTTPError(err) {
		return Retriable
	}

	// По умолчанию считаем ошибку неповторяемой
	return NonRetriable
}

// isNetworkError проверяет, является ли ошибка сетевой (retriable)
func isNetworkError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		// Сетевые ошибки с таймаутом можно повторить
		return netErr.Timeout()
	}

	// Проверяем по тексту ошибки
	errStr := strings.ToLower(err.Error())
	networkKeywords := []string{
		"connection refused",
		"connection reset",
		"connection timeout",
		"no route to host",
		"network is unreachable",
		"temporary failure",
		"timeout",
		"dial tcp",
		"connect: connection refused",
		"i/o timeout",
	}

	for _, keyword := range networkKeywords {
		if strings.Contains(errStr, keyword) {
			return true
		}
	}

	return false
}

// isPostgresRetriableError проверяет, является ли ошибка PostgreSQL retriable
func isPostgresRetriableError(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	// Класс 08 - Ошибки соединения (retriable)
	if pgerrcode.IsConnectionException(pgErr.Code) {
		return true
	}

	// Класс 40 - Откат транзакции (retriable)
	if pgerrcode.IsTransactionRollback(pgErr.Code) {
		return true
	}

	// Конкретные retriable коды ошибок
	switch pgErr.Code {
	case pgerrcode.ConnectionException,           // 08000
		pgerrcode.ConnectionDoesNotExist,         // 08003
		pgerrcode.ConnectionFailure,              // 08006
		pgerrcode.TransactionRollback,            // 40000
		pgerrcode.SerializationFailure,           // 40001
		pgerrcode.DeadlockDetected,               // 40P01
		pgerrcode.CannotConnectNow:               // 57P03
		return true
	}

	return false
}

// isTemporaryHTTPError проверяет, является ли ошибка HTTP временной (retriable)
func isTemporaryHTTPError(err error) bool {
	errStr := strings.ToLower(err.Error())
	
	// HTTP статус коды, которые можно повторить
	temporaryStatusCodes := []string{
		"502 bad gateway",
		"503 service unavailable",
		"504 gateway timeout",
		"429 too many requests",
	}

	for _, statusCode := range temporaryStatusCodes {
		if strings.Contains(errStr, statusCode) {
			return true
		}
	}

	return false
}
