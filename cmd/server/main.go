package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/Mihklz/metrixcollector/internal/config"
	"github.com/Mihklz/metrixcollector/internal/handler"
	"github.com/Mihklz/metrixcollector/internal/logger"
	"github.com/Mihklz/metrixcollector/internal/middleware"
	"github.com/Mihklz/metrixcollector/internal/repository"
)

func main() {
	// Инициализируем логгер
	if err := logger.Initialize(); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	// Обязательно сбрасываем буферы логгера при завершении программы
	defer logger.Log.Sync()

	cfg := config.LoadServerConfig()

	storage := repository.NewMemStorage()

	// Инициализируем chi-роутер
	r := chi.NewRouter()

	// Добавляем middleware для логирования ко всем роутам
	r.Use(func(next http.Handler) http.Handler {
		return logger.WithLogging(next)
	})

	// Добавляем middleware для gzip сжатия/декомпрессии
	r.Use(middleware.WithGzip)

	// === Старые URL-based эндпоинты (для совместимости) ===
	// POST /update/{type}/{name}/{value}
	r.Post("/update/{type}/{name}/{value}", handler.NewUpdateHandler(storage))

	// GET /value/{type}/{name}
	r.Get("/value/{type}/{name}", handler.NewValueHandler(storage))

	// GET / - главная страница со всеми метриками
	r.Get("/", handler.NewRootHandler(storage))

	// === Новые JSON API эндпоинты ===
	// POST /update - принимает метрики в JSON формате
	r.Post("/update", handler.NewJSONUpdateHandler(storage))
	r.Post("/update/", handler.NewJSONUpdateHandler(storage))

	// POST /value - возвращает метрики в JSON формате
	r.Post("/value", handler.NewJSONValueHandler(storage))
	r.Post("/value/", handler.NewJSONValueHandler(storage))

	// Логируем запуск сервера
	logger.Log.Info("Starting metrics collector server",
		zap.String("address", cfg.RunAddr),
	)

	// Запускаем сервер
	if err := http.ListenAndServe(cfg.RunAddr, r); err != nil {
		logger.Log.Fatal("Server failed to start",
			zap.Error(err),
			zap.String("address", cfg.RunAddr),
		)
	}
}
