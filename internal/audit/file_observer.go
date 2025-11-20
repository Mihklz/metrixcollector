package audit

import (
	"os"
	"sync"

	"go.uber.org/zap"

	"github.com/Mihklz/metrixcollector/internal/logger"
)

// FileAuditObserver реализует Observer для записи событий аудита в файл.
type FileAuditObserver struct {
	filePath string
	mu       sync.Mutex
}

// NewFileAuditObserver создает новый наблюдатель для записи в файл.
func NewFileAuditObserver(filePath string) *FileAuditObserver {
	return &FileAuditObserver{
		filePath: filePath,
	}
}

// Notify записывает событие аудита в файл.
func (f *FileAuditObserver) Notify(event *AuditEvent) error {
	// Преобразуем событие в JSON вне критической секции
	data, err := event.ToJSON()
	if err != nil {
		if logger.Log != nil {
			logger.Log.Error("Failed to marshal audit event to JSON",
				zap.Error(err),
			)
		}
		return err
	}

	// Добавляем новую строку после JSON
	data = append(data, '\n')

	// Критическая секция: только операция записи в файл
	f.mu.Lock()
	file, err := os.OpenFile(f.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		f.mu.Unlock()
		if logger.Log != nil {
			logger.Log.Error("Failed to open audit file",
				zap.String("file", f.filePath),
				zap.Error(err),
			)
		}
		return err
	}

	_, err = file.Write(data)
	file.Close()
	f.mu.Unlock()

	if err != nil {
		if logger.Log != nil {
			logger.Log.Error("Failed to write audit event to file",
				zap.String("file", f.filePath),
				zap.Error(err),
			)
		}
		return err
	}

	if logger.Log != nil {
		logger.Log.Debug("Audit event written to file",
			zap.String("file", f.filePath),
			zap.Int64("timestamp", event.Timestamp),
		)
	}

	return nil
}
