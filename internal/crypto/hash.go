package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// CalculateHMAC вычисляет HMAC-SHA256 хеш для данных с использованием ключа
func CalculateHMAC(data []byte, key string) string {
	if key == "" {
		return ""
	}

	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// ValidateHMAC проверяет соответствие HMAC-SHA256 хеша
func ValidateHMAC(data []byte, key string, signature string) bool {
	if key == "" || signature == "" {
		return key == "" && signature == ""
	}

	expectedSignature := CalculateHMAC(data, key)
	return hmac.Equal([]byte(expectedSignature), []byte(signature))
}
