package app

import (
	_ "github.com/lib/pq" // PostgreSQL драйвер
	"go.uber.org/zap"

	"github.com/Mihklz/metrixcollector/internal/config"
	"github.com/Mihklz/metrixcollector/internal/logger"
	"github.com/Mihklz/metrixcollector/internal/repository"
	"github.com/Mihklz/metrixcollector/internal/server"
	"github.com/Mihklz/metrixcollector/internal/service"
)

// ProvideConfig предоставляет конфигурацию сервера
func ProvideConfig() *config.ServerConfig {
	return config.LoadServerConfig()
}

// ProvideDatabase предоставляет подключение к базе данных
func ProvideDatabase(cfg *config.ServerConfig) repository.Database {
	if cfg.DatabaseDSN == "" {
		logger.Log.Info("Database DSN not provided, running without database connection")
		return nil
	}

	logger.Log.Info("Connecting to database", zap.String("dsn", cfg.DatabaseDSN))
	db, err := repository.NewPostgresDB(cfg.DatabaseDSN)
	if err != nil {
		logger.Log.Error("Failed to connect to database, will fallback to file/memory storage", zap.Error(err))
		return nil
	}

	logger.Log.Info("Successfully connected to database")
	return db
}

// ProvideStorage предоставляет хранилище метрик с приоритетом: PostgreSQL -> файл -> память
func ProvideStorage(cfg *config.ServerConfig, db repository.Database) repository.Storage {
	// Приоритет 1: PostgreSQL (если есть подключение к БД)
	if db != nil {
		logger.Log.Info("Using PostgreSQL storage")
		postgresStorage, err := repository.NewPostgresStorage(db.GetConnection(), "migrations")
		if err != nil {
			logger.Log.Error("Failed to create PostgreSQL storage, falling back to file/memory storage",
				zap.Error(err),
			)
		} else {
			return postgresStorage
		}
	}

	// Приоритет 2: Файловое хранилище (если не используется PostgreSQL)
	storage := repository.NewMemStorage()

	// Загружаем метрики при старте, если это включено
	if cfg.Restore {
		logger.Log.Info("Loading metrics from file",
			zap.String("file", cfg.FileStoragePath),
		)

		if err := storage.LoadFromFile(cfg.FileStoragePath); err != nil {
			logger.Log.Error("Failed to load metrics from file",
				zap.Error(err),
				zap.String("file", cfg.FileStoragePath),
			)
		} else {
			logger.Log.Info("Metrics loaded successfully from file")
		}
	}

	// Приоритет 3: Память (по умолчанию)
	logger.Log.Info("Using memory storage with file backup")
	return storage
}

// ProvideFileStorageService предоставляет сервис файлового хранения
func ProvideFileStorageService(cfg *config.ServerConfig, storage repository.Storage) *service.FileStorageService {
	// Если используется PostgreSQL хранилище, файловое сохранение не нужно
	if _, isPostgres := storage.(*repository.PostgresStorage); isPostgres {
		logger.Log.Info("PostgreSQL storage detected, file storage service will be limited")
		// Создаем сервис, но он не будет реально использоваться для PostgreSQL
		return service.NewFileStorageService(
			storage,
			cfg.FileStoragePath,
			0, // Отключаем периодическое сохранение
		)
	}

	return service.NewFileStorageService(
		storage,
		cfg.FileStoragePath,
		cfg.StoreInterval,
	)
}

// ProvideServer предоставляет HTTP сервер
func ProvideServer(cfg *config.ServerConfig, baseStorage repository.Storage, fileService *service.FileStorageService, db repository.Database) *server.Server {
	var storage = baseStorage

	// Если интервал равен 0 и НЕ используется PostgreSQL, используем синхронное сохранение
	if cfg.StoreInterval == 0 {
		if _, isPostgres := baseStorage.(*repository.PostgresStorage); !isPostgres {
			logger.Log.Info("Using synchronous file storage")
			storage = &SyncStorageWithDI{
				Storage:     baseStorage,
				fileService: fileService,
			}
		} else {
			logger.Log.Info("PostgreSQL storage detected, synchronous file storage disabled")
		}
	}

	// Важно: если используется PostgreSQL хранилище, но db == nil, это означает ошибку в логике
	if _, isPostgres := baseStorage.(*repository.PostgresStorage); isPostgres && db == nil {
		logger.Log.Warn("PostgreSQL storage is used but Database object is nil - this should not happen")
	}

	return server.NewServer(cfg, storage, fileService, db)
}

// SyncStorageWithDI - синхронное хранилище с Dependency Injection
type SyncStorageWithDI struct {
	repository.Storage
	fileService *service.FileStorageService
}

// Update выполняет обновление метрики и синхронное сохранение
func (s *SyncStorageWithDI) Update(metricType, name, value string) error {
	// Выполняем обновление
	err := s.Storage.Update(metricType, name, value)
	if err != nil {
		return err
	}

	// Синхронное сохранение в файл через внедренный сервис
	_ = s.fileService.SaveSync() // игнорируем ошибку для fail-safe работы

	return nil
}
