package audit

import (
	"sync"

	"go.uber.org/zap"

	"github.com/Mihklz/metrixcollector/internal/logger"
)

// AuditPublisher реализует Publisher для управления наблюдателями.
type AuditPublisher struct {
	observers []Observer
	mu        sync.RWMutex
}

// NewAuditPublisher создает новый экземпляр издателя событий аудита.
func NewAuditPublisher() *AuditPublisher {
	return &AuditPublisher{
		observers: make([]Observer, 0),
	}
}

// Subscribe добавляет наблюдателя в список подписчиков.
func (p *AuditPublisher) Subscribe(observer Observer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.observers = append(p.observers, observer)
}

// Unsubscribe удаляет наблюдателя из списка подписчиков.
func (p *AuditPublisher) Unsubscribe(observer Observer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, obs := range p.observers {
		if obs == observer {
			p.observers = append(p.observers[:i], p.observers[i+1:]...)
			break
		}
	}
}

// Publish отправляет событие всем подписчикам.
// Выполняется асинхронно, чтобы не блокировать основную обработку запросов.
func (p *AuditPublisher) Publish(event *AuditEvent) {
	p.mu.RLock()
	// Запускаем горутины для каждого наблюдателя
	for _, observer := range p.observers {
		obs := observer // захватываем observer в локальную переменную для замыкания
		go func() {
			if err := obs.Notify(event); err != nil {
				if logger.Log != nil {
					logger.Log.Error("Failed to notify audit observer",
						zap.Error(err),
					)
				}
			}
		}()
	}
	p.mu.RUnlock()
}

// HasObservers проверяет, есть ли зарегистрированные наблюдатели.
func (p *AuditPublisher) HasObservers() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.observers) > 0
}
