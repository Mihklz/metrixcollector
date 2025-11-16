package handler

import (
	"net/http"

	"github.com/Mihklz/metrixcollector/internal/crypto"
)

// WriteResponseWithHash записывает ответ с добавлением хеша в заголовок, если есть ключ
func WriteResponseWithHash(w http.ResponseWriter, data []byte, key string, statusCode int, contentType string) {
	// Добавляем хеш в заголовок, если есть ключ
	if key != "" && len(data) > 0 {
		hash := crypto.CalculateHMAC(data, key)
		w.Header().Set("HashSHA256", hash)
	}

	// Устанавливаем Content-Type, если указан
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}

	// Устанавливаем статус код
	w.WriteHeader(statusCode)

	// Записываем данные
	if len(data) > 0 {
		w.Write(data)
	}
}
