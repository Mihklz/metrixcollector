package audit

import (
	"net"
	"net/http"
	"strings"
)

// GetIPAddress извлекает IP-адрес из HTTP-запроса.
// Учитывает заголовки X-Forwarded-For и X-Real-IP для работы за прокси.
func GetIPAddress(r *http.Request) string {
	// Проверяем заголовок X-Forwarded-For (может содержать несколько IP через запятую)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Берем первый IP из списка
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if ip != "" {
				return ip
			}
		}
	}

	// Проверяем заголовок X-Real-IP
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Используем RemoteAddr как fallback
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return ip
}
