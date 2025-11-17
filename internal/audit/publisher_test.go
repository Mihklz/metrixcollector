package audit

import (
	"sync"
	"testing"
	"time"
)

// TestAuditPublisherMultipleObservers проверяет работу с несколькими наблюдателями
func TestAuditPublisherMultipleObservers(t *testing.T) {
	publisher := NewAuditPublisher()

	observer1 := &mockObserver{events: make([]*AuditEvent, 0)}
	observer2 := &mockObserver{events: make([]*AuditEvent, 0)}
	observer3 := &mockObserver{events: make([]*AuditEvent, 0)}

	publisher.Subscribe(observer1)
	publisher.Subscribe(observer2)
	publisher.Subscribe(observer3)

	event := NewAuditEvent([]string{"TestMetric"}, "127.0.0.1")
	publisher.Publish(event)

	// Даем время на обработку
	time.Sleep(150 * time.Millisecond)

	// Проверяем, что все наблюдатели получили событие
	if len(observer1.events) != 1 {
		t.Errorf("Observer1: expected 1 event, got %d", len(observer1.events))
	}
	if len(observer2.events) != 1 {
		t.Errorf("Observer2: expected 1 event, got %d", len(observer2.events))
	}
	if len(observer3.events) != 1 {
		t.Errorf("Observer3: expected 1 event, got %d", len(observer3.events))
	}
}

// TestAuditPublisherUnsubscribeSpecific проверяет отписку конкретного наблюдателя
func TestAuditPublisherUnsubscribeSpecific(t *testing.T) {
	publisher := NewAuditPublisher()

	observer1 := &mockObserver{events: make([]*AuditEvent, 0)}
	observer2 := &mockObserver{events: make([]*AuditEvent, 0)}
	observer3 := &mockObserver{events: make([]*AuditEvent, 0)}

	publisher.Subscribe(observer1)
	publisher.Subscribe(observer2)
	publisher.Subscribe(observer3)

	// Отписываем второго наблюдателя
	publisher.Unsubscribe(observer2)

	event := NewAuditEvent([]string{"TestMetric"}, "127.0.0.1")
	publisher.Publish(event)

	time.Sleep(150 * time.Millisecond)

	// Проверяем, что observer2 не получил событие
	if len(observer1.events) != 1 {
		t.Errorf("Observer1: expected 1 event, got %d", len(observer1.events))
	}
	if len(observer2.events) != 0 {
		t.Errorf("Observer2: expected 0 events (unsubscribed), got %d", len(observer2.events))
	}
	if len(observer3.events) != 1 {
		t.Errorf("Observer3: expected 1 event, got %d", len(observer3.events))
	}
}

// TestAuditPublisherConcurrentPublish проверяет конкурентную публикацию событий
func TestAuditPublisherConcurrentPublish(t *testing.T) {
	publisher := NewAuditPublisher()
	observer := &threadSafeMockObserver{
		events: make([]*AuditEvent, 0),
	}
	publisher.Subscribe(observer)

	numEvents := 100
	var wg sync.WaitGroup
	wg.Add(numEvents)

	// Публикуем события конкурентно
	for i := 0; i < numEvents; i++ {
		go func(idx int) {
			defer wg.Done()
			event := NewAuditEvent([]string{"Metric" + string(rune(idx))}, "127.0.0.1")
			publisher.Publish(event)
		}(i)
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	observer.mu.Lock()
	receivedCount := len(observer.events)
	observer.mu.Unlock()

	if receivedCount != numEvents {
		t.Errorf("Expected %d events, got %d", numEvents, receivedCount)
	}
}

// TestAuditPublisherConcurrentSubscribe проверяет конкурентную подписку
func TestAuditPublisherConcurrentSubscribe(t *testing.T) {
	publisher := NewAuditPublisher()

	numObservers := 50
	var wg sync.WaitGroup
	wg.Add(numObservers)

	observers := make([]*mockObserver, numObservers)
	for i := 0; i < numObservers; i++ {
		observers[i] = &mockObserver{events: make([]*AuditEvent, 0)}
	}

	// Подписываем наблюдателей конкурентно
	for i := 0; i < numObservers; i++ {
		go func(idx int) {
			defer wg.Done()
			publisher.Subscribe(observers[idx])
		}(i)
	}

	wg.Wait()

	if !publisher.HasObservers() {
		t.Error("Publisher should have observers")
	}

	// Публикуем событие и проверяем, что все получили его
	event := NewAuditEvent([]string{"TestMetric"}, "127.0.0.1")
	publisher.Publish(event)

	time.Sleep(200 * time.Millisecond)

	receivedCount := 0
	for _, obs := range observers {
		if len(obs.events) > 0 {
			receivedCount++
		}
	}

	if receivedCount != numObservers {
		t.Errorf("Expected %d observers to receive event, got %d", numObservers, receivedCount)
	}
}

// TestAuditPublisherErrorHandling проверяет обработку ошибок от наблюдателей
func TestAuditPublisherErrorHandling(t *testing.T) {
	publisher := NewAuditPublisher()

	// Наблюдатель, который всегда возвращает ошибку
	errorObserver := &errorMockObserver{}
	normalObserver := &mockObserver{events: make([]*AuditEvent, 0)}

	publisher.Subscribe(errorObserver)
	publisher.Subscribe(normalObserver)

	event := NewAuditEvent([]string{"TestMetric"}, "127.0.0.1")
	publisher.Publish(event)

	time.Sleep(150 * time.Millisecond)

	// Проверяем, что нормальный наблюдатель всё равно получил событие
	if len(normalObserver.events) != 1 {
		t.Errorf("Normal observer should receive event despite error in other observer, got %d events", len(normalObserver.events))
	}
}

// TestAuditPublisherNoObservers проверяет поведение без наблюдателей
func TestAuditPublisherNoObservers(t *testing.T) {
	publisher := NewAuditPublisher()

	if publisher.HasObservers() {
		t.Error("New publisher should not have observers")
	}

	// Публикация без наблюдателей не должна вызывать панику
	event := NewAuditEvent([]string{"TestMetric"}, "127.0.0.1")
	publisher.Publish(event)

	// Даем время на возможную обработку
	time.Sleep(50 * time.Millisecond)

	// Тест проходит, если не было паники
}

// Вспомогательные типы для тестов

type threadSafeMockObserver struct {
	events []*AuditEvent
	mu     sync.Mutex
}

func (m *threadSafeMockObserver) Notify(event *AuditEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}

type errorMockObserver struct{}

func (m *errorMockObserver) Notify(event *AuditEvent) error {
	return &testError{msg: "intentional test error"}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

