package handler

import (
	"github.com/Mihklz/metrixcollector/internal/repository"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUpdateHandler_ValidGauge(t *testing.T) {
	store := repository.NewMemStorage()
	handler := NewUpdateHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/TestGauge/123.4", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	if val := store.Gauges["TestGauge"]; val != 123.4 {
		t.Errorf("expected value 123.4, got %v", val)
	}
}

func TestUpdateHandler_UnsupportedMethod(t *testing.T) {
	store := repository.NewMemStorage()
	handler := NewUpdateHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/update/gauge/TestGauge/123.4", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Error("expected 405 Method Not Allowed")
	}
}

func TestUpdateHandler_InvalidPath(t *testing.T) {
	store := repository.NewMemStorage()
	handler := NewUpdateHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/update/gaugeonly", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Result().StatusCode != http.StatusNotFound {
		t.Error("expected 404 Not Found")
	}
}
