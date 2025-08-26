package app

import (
	"go.uber.org/zap"

	"github.com/Mihklz/metrixcollector/internal/config"
	"github.com/Mihklz/metrixcollector/internal/logger"
	"github.com/Mihklz/metrixcollector/internal/repository"
	"github.com/Mihklz/metrixcollector/internal/server"
	"github.com/Mihklz/metrixcollector/internal/service"
)

// AppFactory отвечает за создание и конфигурирование компонентов приложения
type AppFactory struct {
	config *config.ServerConfig
}

// NewAppFactory создает новую фабрику приложения
func NewAppFactory(cfg *config.ServerConfig) *AppFactory {
	return &AppFactory{
		config: cfg,
	}
}

// CreateStorage создает и конфигурирует хранилище метрик
func (f *AppFactory) CreateStorage() repository.Storage {
	storage := repository.NewMemStorage()

	// Загружаем метрики при старте, если это включено
	if f.config.Restore {
		logger.Log.Info("Loading metrics from file",
			zap.String("file", f.config.FileStoragePath),
		)

		if err := storage.LoadFromFile(f.config.FileStoragePath); err != nil {
			logger.Log.Error("Failed to load metrics from file",
				zap.Error(err),
				zap.String("file", f.config.FileStoragePath),
			)
		} else {
			logger.Log.Info("Metrics loaded successfully from file")
		}
	}

	return storage
}

// CreateFileStorageService создает сервис файлового хранения
func (f *AppFactory) CreateFileStorageService(storage repository.Storage) *service.FileStorageService {
	return service.NewFileStorageService(
		storage,
		f.config.FileStoragePath,
		f.config.StoreInterval,
	)
}

// CreateSyncStorage создает синхронное хранилище (если StoreInterval = 0)
func (f *AppFactory) CreateSyncStorage(baseStorage repository.Storage, fileService *service.FileStorageService) repository.Storage {
	if f.config.StoreInterval == 0 {
		logger.Log.Info("Using synchronous file storage")
		return NewSyncStorage(baseStorage, fileService)
	}

	return baseStorage
}

// CreateServer создает HTTP сервер со всеми зависимостями
func (f *AppFactory) CreateServer(storage repository.Storage, fileService *service.FileStorageService) *server.Server {
	return server.NewServer(f.config, storage, fileService)
}

// SyncStorage оборачивает storage для синхронного сохранения
type SyncStorage struct {
	repository.Storage
	fileService *service.FileStorageService
}

// NewSyncStorage создает новое синхронное хранилище
func NewSyncStorage(storage repository.Storage, fileService *service.FileStorageService) *SyncStorage {
	return &SyncStorage{
		Storage:     storage,
		fileService: fileService,
	}
}

// Update выполняет обновление метрики и синхронное сохранение
func (s *SyncStorage) Update(metricType, name, value string) error {
	// Выполняем обновление
	err := s.Storage.Update(metricType, name, value)
	if err != nil {
		return err
	}

	// Синхронное сохранение в файл через сервис
	_ = s.fileService.SaveSync() // игнорируем ошибку для fail-safe работы

	return nil
}
