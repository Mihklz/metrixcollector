package audit

// Observer представляет наблюдателя, который получает события аудита
type Observer interface {
	// Notify отправляет событие аудита наблюдателю
	Notify(event *AuditEvent) error
}

// Publisher управляет списком наблюдателей и отправляет им события
type Publisher interface {
	// Subscribe добавляет наблюдателя в список подписчиков
	Subscribe(observer Observer)
	// Unsubscribe удаляет наблюдателя из списка подписчиков
	Unsubscribe(observer Observer)
	// Publish отправляет событие всем подписчикам
	Publish(event *AuditEvent)
}

