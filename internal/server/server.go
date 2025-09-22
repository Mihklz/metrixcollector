package server

import (
	"context"
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
	"github.com/Mihklz/metrixcollector/internal/service"
)

// Server представляет HTTP сервер для сбора метрик
type Server struct {
	config         *config.ServerConfig
	storage        repository.Storage
	fileService    *service.FileStorageService
	metricsService *service.MetricsService
	db             repository.Database
	httpServer     *http.Server
	router         *chi.Mux
}

// NewServer создает новый экземпляр сервера
func NewServer(cfg *config.ServerConfig, storage repository.Storage, fileService *service.FileStorageService, db repository.Database) *Server {
	metricsService := service.NewMetricsService(storage)

	server := &Server{
		config:         cfg,
		storage:        storage,
		fileService:    fileService,
		metricsService: metricsService,
		db:             db,
	}

	server.setupRouter()
	server.setupHTTPServer()

	return server
}

// setupRouter настраивает маршруты
func (s *Server) setupRouter() {
	r := chi.NewRouter()

	// Добавляем middleware
	r.Use(func(next http.Handler) http.Handler {
		return logger.WithLogging(next)
	})
	r.Use(middleware.WithGzip)

	// === Старые URL-based эндпоинты (для совместимости) ===
	r.Post("/update/{type}/{name}/{value}", handler.NewUpdateHandler(s.storage))
	r.Get("/value/{type}/{name}", handler.NewValueHandler(s.storage))
	r.Get("/", handler.NewRootHandler(s.storage))

	// === Новые JSON API эндпоинты ===
	r.Post("/update", handler.NewJSONUpdateHandler(s.storage))
	r.Post("/update/", handler.NewJSONUpdateHandler(s.storage))
	r.Post("/value", handler.NewJSONValueHandler(s.storage))
	r.Post("/value/", handler.NewJSONValueHandler(s.storage))

	// === Batch API эндпоинт ===
	r.Post("/updates/", handler.NewBatchUpdateHandler(s.metricsService))

	// === Эндпоинт для проверки соединения с БД ===
	// Если используется PostgreSQL хранилище, создаем новый Database объект для ping
	var pingDB repository.Database = s.db
	if postgresStorage, isPostgres := s.storage.(*repository.PostgresStorage); isPostgres {
		// Для PostgreSQL хранилища создаем Database объект из соединения
		if conn := postgresStorage.GetConnection(); conn != nil {
			pingDB = &repository.PostgresDB{DB: conn}
		}
	}
	r.Get("/ping", handler.NewPingHandler(pingDB))

	s.router = r
}

// setupHTTPServer настраивает HTTP сервер
func (s *Server) setupHTTPServer() {
	s.httpServer = &http.Server{
		Addr:    s.config.RunAddr,
		Handler: s.router,
	}
}

// Run запускает сервер и блокирует выполнение до получения сигнала завершения
func (s *Server) Run() error {
	// Создаём контекст для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запускаем периодическое сохранение, если интервал > 0
	if s.config.StoreInterval > 0 {
		go s.fileService.StartPeriodicSave(ctx)
	}

	// Канал для получения сигналов ОС
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Запускаем сервер в горутине
	go func() {
		logger.Log.Info("Starting metrics collector server",
			zap.String("address", s.config.RunAddr),
			zap.Int("store_interval", s.config.StoreInterval),
			zap.String("file_storage_path", s.config.FileStoragePath),
			zap.Bool("restore", s.config.Restore),
		)

		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Server failed to start",
				zap.Error(err),
				zap.String("address", s.config.RunAddr),
			)
		}
	}()

	// Ожидаем сигнал для завершения
	<-quit
	logger.Log.Info("Shutdown signal received")

	return s.Shutdown(ctx)
}

// Shutdown выполняет graceful shutdown сервера
func (s *Server) Shutdown(ctx context.Context) error {
	// Завершаем периодическое сохранение
	// (ctx отменится, что остановит горутину fileService)

	// Выполняем финальное сохранение метрик
	if err := s.fileService.SaveSync(); err != nil {
		logger.Log.Error("Failed to save metrics on shutdown", zap.Error(err))
	} else {
		logger.Log.Info("Metrics saved successfully on shutdown")
	}

	// Graceful shutdown HTTP сервера
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Log.Error("Server forced to shutdown", zap.Error(err))
		return err
	}

	logger.Log.Info("Server stopped gracefully")
	return nil
}
