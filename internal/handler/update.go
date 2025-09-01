package handler

import (
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/Mihklz/metrixcollector/internal/logger"
	"github.com/Mihklz/metrixcollector/internal/repository"
)

func NewUpdateHandler(storage repository.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/update/"), "/")
		if len(parts) != 3 {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		metricType, name, value := parts[0], parts[1], parts[2]

		err := storage.Update(metricType, name, value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		logger.Log.Info("Metric received and saved",
			zap.String("type", metricType),
			zap.String("name", name),
			zap.String("value", value),
		)
		w.WriteHeader(http.StatusOK)
	}
}
