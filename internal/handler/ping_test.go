package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/Mihklz/metrixcollector/internal/logger"
	"github.com/Mihklz/metrixcollector/internal/repository/mocks"
)

func init() {
	// Инициализируем логгер для тестов
	logger.Log = zap.NewNop() // Используем no-op логгер для тестов
}

func TestPingHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		dbMock         func(ctrl *gomock.Controller) *mocks.MockDatabase
		expectedStatus int
	}{
		{
			name:   "successful ping",
			method: http.MethodGet,
			dbMock: func(ctrl *gomock.Controller) *mocks.MockDatabase {
				db := mocks.NewMockDatabase(ctrl)
				db.EXPECT().Ping(gomock.Any()).Return(nil)
				return db
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "database ping fails",
			method: http.MethodGet,
			dbMock: func(ctrl *gomock.Controller) *mocks.MockDatabase {
				db := mocks.NewMockDatabase(ctrl)
				db.EXPECT().Ping(gomock.Any()).Return(errors.New("connection failed"))
				return db
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "wrong method",
			method: http.MethodPost,
			dbMock: func(ctrl *gomock.Controller) *mocks.MockDatabase {
				db := mocks.NewMockDatabase(ctrl)
				// Не ожидаем вызова Ping для неправильного метода
				return db
			},
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			db := tt.dbMock(ctrl)
			handler := NewPingHandler(db)

			req := httptest.NewRequest(tt.method, "/ping", nil)
			req = req.WithContext(context.Background())
			w := httptest.NewRecorder()

			handler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestPingHandlerWithNilDatabase(t *testing.T) {
	handler := NewPingHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req = req.WithContext(context.Background())
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPingHandlerWithContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Создаем контекст с отменой
	ctx, cancel := context.WithCancel(context.Background())

	db := mocks.NewMockDatabase(ctrl)
	// Проверяем, что контекст передается в метод Ping
	db.EXPECT().Ping(ctx).Return(nil)

	handler := NewPingHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	cancel() // Отменяем контекст после теста
}
