package app

import (
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

// ProvideStorage предоставляет базовое хранилище метрик
func ProvideStorage(cfg *config.ServerConfig) repository.Storage {
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

	return storage
}

// ProvideFileStorageService предоставляет сервис файлового хранения
func ProvideFileStorageService(cfg *config.ServerConfig, storage repository.Storage) *service.FileStorageService {
	return service.NewFileStorageService(
		storage,
		cfg.FileStoragePath,
		cfg.StoreInterval,
	)
}

// ProvideServer предоставляет HTTP сервер
func ProvideServer(cfg *config.ServerConfig, baseStorage repository.Storage, fileService *service.FileStorageService) *server.Server {
	var storage repository.Storage = baseStorage

	// Если интервал равен 0, используем синхронное сохранение
	if cfg.StoreInterval == 0 {
		logger.Log.Info("Using synchronous file storage")
		storage = &SyncStorageWithDI{
			Storage:     baseStorage,
			fileService: fileService,
		}
	}

	return server.NewServer(cfg, storage, fileService)
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
	if saveErr := s.fileService.SaveSync(); saveErr != nil {
		// Логирование уже внутри fileService.SaveSync()
		// Не возвращаем ошибку, чтобы не прерывать работу
	}

	return nil
}
