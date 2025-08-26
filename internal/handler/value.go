package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	models "github.com/Mihklz/metrixcollector/internal/model"
	"github.com/Mihklz/metrixcollector/internal/repository"
)

func NewValueHandler(storage repository.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metricType := chi.URLParam(r, "type")
		metricName := chi.URLParam(r, "name")

		switch metricType {
		case models.Gauge:
			if val, ok := storage.GetGauge(metricName); ok {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(strconv.FormatFloat(float64(val), 'f', -1, 64)))
				return
			}
			http.NotFound(w, r)
		case models.Counter:
			if val, ok := storage.GetCounter(metricName); ok {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(strconv.FormatInt(int64(val), 10)))
				return
			}
			http.NotFound(w, r)
		default:
			http.Error(w, "unsupported metric type", http.StatusNotFound)
		}
	}
}
