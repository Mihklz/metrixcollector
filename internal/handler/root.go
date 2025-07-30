package handler

import (
	"fmt"
	"github.com/Mihklz/metrixcollector/internal/repository"
	"net/http"
	"sort"
)

func NewRootHandler(storage repository.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		// Начинаем HTML-страницу
		_, _ = w.Write([]byte("<html><head><title>Metrics</title></head><body><h1>All Metrics</h1><ul>"))

		// Получаем метрики
		gauges := storage.GetAllGauges()
		counters := storage.GetAllCounters()

		// Чтобы было красиво — отсортируем ключи
		var gaugeNames, counterNames []string
		for k := range gauges {
			gaugeNames = append(gaugeNames, k)
		}
		for k := range counters {
			counterNames = append(counterNames, k)
		}
		sort.Strings(gaugeNames)
		sort.Strings(counterNames)

		// Выводим gauges
		for _, name := range gaugeNames {
			_, _ = fmt.Fprintf(w, "<li>gauge %s = %f</li>", name, gauges[name])
		}

		// Выводим counters
		for _, name := range counterNames {
			_, _ = fmt.Fprintf(w, "<li>counter %s = %d</li>", name, counters[name])
		}

		// Заканчиваем HTML
		_, _ = w.Write([]byte("</ul></body></html>"))
	}
}
