package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/Mihklz/metrixcollector/internal/config"
	"github.com/Mihklz/metrixcollector/internal/handler"
	"github.com/Mihklz/metrixcollector/internal/logger"
	"github.com/Mihklz/metrixcollector/internal/middleware"
	"github.com/Mihklz/metrixcollector/internal/repository"
)

// startPeriodicSave запускает периодическое сохранение метрик
func startPeriodicSave(ctx context.Context, storage repository.Storage, filePath string, intervalSeconds int) {
	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	defer ticker.Stop()

	logger.Log.Info("Started periodic metrics saving",
		zap.Int("interval_seconds", intervalSeconds),
		zap.String("file_path", filePath),
	)

	for {
		select {
		case <-ctx.Done():
			logger.Log.Info("Periodic save stopped")
			return
		case <-ticker.C:
			logger.Log.Debug("Performing periodic save")
			if err := storage.SaveToFile(filePath); err != nil {
				logger.Log.Error("Failed to save metrics periodically",
					zap.Error(err),
					zap.String("file_path", filePath),
				)
			} else {
				logger.Log.Debug("Metrics saved successfully",
					zap.String("file_path", filePath),
				)
			}
		}
	}
}

// syncStorage оборачивает storage для синхронного сохранения
type syncStorage struct {
	repository.Storage
	filePath string
}

func (s *syncStorage) Update(metricType, name, value string) error {
	// Выполняем обновление
	err := s.Storage.Update(metricType, name, value)
	if err != nil {
		return err
	}

	// Синхронное сохранение в файл
	if saveErr := s.Storage.SaveToFile(s.filePath); saveErr != nil {
		logger.Log.Error("Failed to save metrics synchronously",
			zap.Error(saveErr),
			zap.String("file_path", s.filePath),
		)
	}

	return nil
}

func main() {
	// Инициализируем логгер
	if err := logger.Initialize(); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	// Обязательно сбрасываем буферы логгера при завершении программы
	defer logger.Log.Sync()

	cfg := config.LoadServerConfig()

	baseStorage := repository.NewMemStorage()

	// Загружаем метрики при старте, если это включено
	if cfg.Restore {
		logger.Log.Info("Loading metrics from file",
			zap.String("file", cfg.FileStoragePath),
		)
		if err := baseStorage.LoadFromFile(cfg.FileStoragePath); err != nil {
			logger.Log.Error("Failed to load metrics from file",
				zap.Error(err),
				zap.String("file", cfg.FileStoragePath),
			)
		} else {
			logger.Log.Info("Metrics loaded successfully from file")
		}
	}

	// Определяем, какой storage использовать
	var storage repository.Storage = baseStorage

	// Если интервал равен 0, используем синхронное сохранение
	if cfg.StoreInterval == 0 {
		storage = &syncStorage{
			Storage:  baseStorage,
			filePath: cfg.FileStoragePath,
		}
		logger.Log.Info("Using synchronous file storage")
	}

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

	// Создаём HTTP сервер
	server := &http.Server{
		Addr:    cfg.RunAddr,
		Handler: r,
	}

	// Создаём контекст для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запускаем периодическое сохранение метрик, если интервал > 0
	if cfg.StoreInterval > 0 {
		go startPeriodicSave(ctx, baseStorage, cfg.FileStoragePath, cfg.StoreInterval)
	}

	// Канал для получения сигналов ОС
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Запускаем сервер в горутине
	go func() {
		logger.Log.Info("Starting metrics collector server",
			zap.String("address", cfg.RunAddr),
			zap.Int("store_interval", cfg.StoreInterval),
			zap.String("file_storage_path", cfg.FileStoragePath),
			zap.Bool("restore", cfg.Restore),
		)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Server failed to start",
				zap.Error(err),
				zap.String("address", cfg.RunAddr),
			)
		}
	}()

	// Ожидаем сигнал для завершения
	<-quit
	logger.Log.Info("Shutdown signal received")

	// Завершаем периодическое сохранение
	cancel()

	// Выполняем финальное сохранение метрик
	logger.Log.Info("Saving metrics before shutdown")
	if err := baseStorage.SaveToFile(cfg.FileStoragePath); err != nil {
		logger.Log.Error("Failed to save metrics on shutdown", zap.Error(err))
	} else {
		logger.Log.Info("Metrics saved successfully on shutdown")
	}

	// Graceful shutdown сервера
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Log.Error("Server forced to shutdown", zap.Error(err))
	} else {
		logger.Log.Info("Server stopped gracefully")
	}
}
