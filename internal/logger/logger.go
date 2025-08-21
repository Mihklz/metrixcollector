package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Log - глобальный логгер для всего приложения (синглтон)
var Log *zap.Logger

// Initialize инициализирует глобальный логгер
// Эта функция должна быть вызвана один раз при старте приложения
func Initialize() error {
	// Создаём production конфигурацию логгера
	config := zap.NewProductionConfig()

	// Устанавливаем уровень логирования на Info
	config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)

	// Создаём логгер на основе конфигурации
	logger, err := config.Build()
	if err != nil {
		return err
	}

	// Устанавливаем глобальный логгер
	Log = logger
	return nil
}

// responseData - структура для хранения данных об HTTP-ответе
// Нужна для перехвата статуса и размера ответа
type responseData struct {
	status int // HTTP статус код
	size   int // размер тела ответа в байтах
}

// loggingResponseWriter - обёртка над стандартным http.ResponseWriter
// Позволяет перехватывать статус код и размер ответа для логирования
type loggingResponseWriter struct {
	http.ResponseWriter               // встраиваем оригинальный http.ResponseWriter
	responseData        *responseData // ссылка на структуру с данными ответа
}

// Write перехватывает запись тела ответа и сохраняет его размер
func (lw *loggingResponseWriter) Write(b []byte) (int, error) {
	// Вызываем оригинальный метод для записи ответа
	size, err := lw.ResponseWriter.Write(b)
	// Сохраняем размер записанных данных (накопительно)
	lw.responseData.size += size
	return size, err
}

// WriteHeader перехватывает установку статус кода ответа
func (lw *loggingResponseWriter) WriteHeader(statusCode int) {
	// Вызываем оригинальный метод для установки статуса
	lw.ResponseWriter.WriteHeader(statusCode)
	// Сохраняем статус код для логирования
	lw.responseData.status = statusCode
}

// WithLogging - middleware для логирования HTTP запросов и ответов
// Оборачивает любой http.Handler и добавляет логирование
func WithLogging(h http.Handler) http.Handler {
	// Возвращаем новый handler, который логирует запросы
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Засекаем время начала обработки запроса
		start := time.Now()

		// Создаём структуру для хранения данных ответа
		responseData := &responseData{
			status: 200, // По умолчанию 200 (если WriteHeader не вызван)
			size:   0,   // Изначально размер 0
		}

		// Создаём нашу обёртку ResponseWriter
		lw := &loggingResponseWriter{
			ResponseWriter: w,
			responseData:   responseData,
		}

		// Вызываем оригинальный handler с нашей обёрткой
		// Теперь все вызовы Write() и WriteHeader() будут перехвачены
		h.ServeHTTP(lw, r)

		// Вычисляем время выполнения запроса
		duration := time.Since(start)

		// Логируем всю информацию о запросе и ответе
		Log.Info("HTTP request processed",
			zap.String("uri", r.RequestURI),        // URI запроса
			zap.String("method", r.Method),         // HTTP метод
			zap.Int("status", responseData.status), // статус ответа
			zap.Duration("duration", duration),     // время выполнения
			zap.Int("size", responseData.size),     // размер ответа
		)
	})
}
