package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/Mihklz/metrixcollector/internal/logger"
	"github.com/Mihklz/metrixcollector/internal/repository"
)

// FileStorageService отвечает за периодическое сохранение метрик в файл
type FileStorageService struct {
	storage  repository.Storage
	filePath string
	interval time.Duration
	logger   *zap.Logger
}

// NewFileStorageService создает новый сервис файлового хранения
func NewFileStorageService(storage repository.Storage, filePath string, intervalSeconds int) *FileStorageService {
	return &FileStorageService{
		storage:  storage,
		filePath: filePath,
		interval: time.Duration(intervalSeconds) * time.Second,
		logger:   logger.Log,
	}
}

// StartPeriodicSave запускает периодическое сохранение метрик
func (fs *FileStorageService) StartPeriodicSave(ctx context.Context) {
	// Если интервал равен 0 или меньше, не запускаем периодическое сохранение
	if fs.interval <= 0 {
		fs.logger.Info("Periodic save disabled (interval <= 0)")
		return
	}
	
	ticker := time.NewTicker(fs.interval)
	defer ticker.Stop()

	fs.logger.Info("Started periodic metrics saving",
		zap.Duration("interval", fs.interval),
		zap.String("file_path", fs.filePath),
	)

	for {
		select {
		case <-ctx.Done():
			fs.logger.Info("Periodic save stopped")
			return
		case <-ticker.C:
			fs.performSave()
		}
	}
}

// SaveSync выполняет синхронное сохранение метрик
func (fs *FileStorageService) SaveSync() error {
	fs.logger.Info("Performing synchronous save", zap.String("file_path", fs.filePath))

	if err := fs.storage.SaveToFile(fs.filePath); err != nil {
		fs.logger.Error("Failed to save metrics synchronously",
			zap.Error(err),
			zap.String("file_path", fs.filePath),
		)
		return err
	}

	fs.logger.Info("Metrics saved successfully", zap.String("file_path", fs.filePath))
	return nil
}

// performSave выполняет сохранение с логированием
func (fs *FileStorageService) performSave() {
	fs.logger.Debug("Performing periodic save")

	if err := fs.storage.SaveToFile(fs.filePath); err != nil {
		fs.logger.Error("Failed to save metrics periodically",
			zap.Error(err),
			zap.String("file_path", fs.filePath),
		)
	} else {
		fs.logger.Debug("Metrics saved successfully",
			zap.String("file_path", fs.filePath),
		)
	}
}
