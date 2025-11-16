package middleware

import (
	"bytes"
	"io"
	"net/http"

	"go.uber.org/zap"

	"github.com/Mihklz/metrixcollector/internal/crypto"
	"github.com/Mihklz/metrixcollector/internal/logger"
)

// WithHashValidation создает middleware для проверки хеша данных
func WithHashValidation(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Если ключ не задан, пропускаем проверку
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Читаем тело запроса
			body, err := io.ReadAll(r.Body)
			if err != nil {
				logger.Log.Error("Failed to read request body", zap.Error(err))
				http.Error(w, "Failed to read request body", http.StatusBadRequest)
				return
			}
			r.Body.Close()

			// Восстанавливаем тело запроса для последующих обработчиков
			r.Body = io.NopCloser(bytes.NewBuffer(body))

			// Получаем хеш из заголовка
			receivedHash := r.Header.Get("HashSHA256")

			// Проверяем хеш только если он присутствует в запросе
			if receivedHash != "" {
				if !crypto.ValidateHMAC(body, key, receivedHash) {
					logger.Log.Warn("Hash validation failed",
						zap.String("received_hash", receivedHash),
						zap.String("method", r.Method),
						zap.String("url", r.URL.Path),
					)
					http.Error(w, "Hash validation failed", http.StatusBadRequest)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
