package handler

import (
	"net/http"

	"github.com/Mihklz/metrixcollector/internal/logger"
	"github.com/Mihklz/metrixcollector/internal/repository"
	"go.uber.org/zap"
)

// PingHandler обрабатывает запросы для проверки соединения с базой данных
type PingHandler struct {
	db repository.Database
}

// NewPingHandler создает новый обработчик для ping
func NewPingHandler(db repository.Database) http.HandlerFunc {
	handler := &PingHandler{db: db}
	return handler.Handle
}

// Handle обрабатывает GET запрос к /ping
func (h *PingHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Если база данных не подключена, возвращаем ошибку
	if h.db == nil {
		logger.Log.Error("Database connection is not initialized")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Проверяем соединение с базой данных
	if err := h.db.Ping(r.Context()); err != nil {
		logger.Log.Error("Database ping failed", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Успешная проверка соединения
	w.WriteHeader(http.StatusOK)
	logger.Log.Debug("Database ping successful")
}
