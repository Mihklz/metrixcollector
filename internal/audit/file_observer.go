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
	f.mu.Lock()
	defer f.mu.Unlock()

	// Преобразуем событие в JSON
	data, err := event.ToJSON()
	if err != nil {
		if logger.Log != nil {
			logger.Log.Error("Failed to marshal audit event to JSON",
				zap.Error(err),
			)
		}
		return err
	}

	// Открываем файл в режиме добавления (создаем, если не существует)
	file, err := os.OpenFile(f.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		if logger.Log != nil {
			logger.Log.Error("Failed to open audit file",
				zap.String("file", f.filePath),
				zap.Error(err),
			)
		}
		return err
	}
	defer file.Close()

	// Добавляем новую строку после JSON
	data = append(data, '\n')

	// Записываем данные в файл
	if _, err := file.Write(data); err != nil {
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
