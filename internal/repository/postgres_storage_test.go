package repository

import (
	"database/sql"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/Mihklz/metrixcollector/internal/logger"
	_ "github.com/lib/pq"
)

func init() {
	// Инициализируем логгер для тестов
	logger.Log = zap.NewNop()
}

// TestPostgresStorage_Integration интеграционный тест для PostgreSQL хранилища
// Требует наличия PostgreSQL БД для тестирования
func TestPostgresStorage_Integration(t *testing.T) {
	// Пропускаем тест, если нет переменной окружения для тестовой БД
	testDSN := os.Getenv("TEST_DATABASE_DSN")
	if testDSN == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping integration test")
	}

	// Подключаемся к тестовой БД
	db, err := sql.Open("postgres", testDSN)
	require.NoError(t, err)
	defer db.Close()

	// Проверяем соединение
	err = db.Ping()
	require.NoError(t, err)

	// Создаем PostgreSQL хранилище
	storage, err := NewPostgresStorage(db, "../../migrations")
	require.NoError(t, err)
	defer storage.Close()

	t.Run("UpdateAndGetGauge", func(t *testing.T) {
		// Обновляем gauge метрику
		err := storage.Update("gauge", "test_gauge", "123.45")
		assert.NoError(t, err)

		// Получаем значение
		value, exists := storage.GetGauge("test_gauge")
		assert.True(t, exists)
		assert.Equal(t, Gauge(123.45), value)

		// Обновляем еще раз
		err = storage.Update("gauge", "test_gauge", "678.90")
		assert.NoError(t, err)

		// Проверяем, что значение изменилось
		value, exists = storage.GetGauge("test_gauge")
		assert.True(t, exists)
		assert.Equal(t, Gauge(678.90), value)
	})

	t.Run("UpdateAndGetCounter", func(t *testing.T) {
		// Обновляем counter метрику
		err := storage.Update("counter", "test_counter", "10")
		assert.NoError(t, err)

		// Получаем значение
		value, exists := storage.GetCounter("test_counter")
		assert.True(t, exists)
		assert.Equal(t, Counter(10), value)

		// Обновляем еще раз (должно прибавиться)
		err = storage.Update("counter", "test_counter", "5")
		assert.NoError(t, err)

		// Проверяем, что значение увеличилось
		value, exists = storage.GetCounter("test_counter")
		assert.True(t, exists)
		assert.Equal(t, Counter(15), value)
	})

	t.Run("GetAllMetrics", func(t *testing.T) {
		// Добавляем несколько метрик
		err := storage.Update("gauge", "gauge1", "1.1")
		assert.NoError(t, err)
		err = storage.Update("gauge", "gauge2", "2.2")
		assert.NoError(t, err)
		err = storage.Update("counter", "counter1", "100")
		assert.NoError(t, err)
		err = storage.Update("counter", "counter2", "200")
		assert.NoError(t, err)

		// Получаем все gauge метрики
		gauges := storage.GetAllGauges()
		assert.Contains(t, gauges, "gauge1")
		assert.Contains(t, gauges, "gauge2")
		assert.Equal(t, Gauge(1.1), gauges["gauge1"])
		assert.Equal(t, Gauge(2.2), gauges["gauge2"])

		// Получаем все counter метрики
		counters := storage.GetAllCounters()
		assert.Contains(t, counters, "counter1")
		assert.Contains(t, counters, "counter2")
		assert.Equal(t, Counter(100), counters["counter1"])
		assert.Equal(t, Counter(200), counters["counter2"])
	})

	t.Run("GetNonExistentMetric", func(t *testing.T) {
		// Проверяем получение несуществующей gauge метрики
		_, exists := storage.GetGauge("non_existent_gauge")
		assert.False(t, exists)

		// Проверяем получение несуществующей counter метрики
		_, exists = storage.GetCounter("non_existent_counter")
		assert.False(t, exists)
	})

	t.Run("InvalidMetricType", func(t *testing.T) {
		// Проверяем обработку неподдерживаемого типа метрики
		err := storage.Update("invalid_type", "test_metric", "123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported metric type")
	})

	t.Run("InvalidValues", func(t *testing.T) {
		// Проверяем обработку некорректных значений для gauge
		err := storage.Update("gauge", "test_gauge", "not_a_number")
		assert.Error(t, err)

		// Проверяем обработку некорректных значений для counter
		err = storage.Update("counter", "test_counter", "not_a_number")
		assert.Error(t, err)
	})

	t.Run("FileOperationsNotSupported", func(t *testing.T) {
		// Проверяем, что файловые операции не поддерживаются
		err := storage.SaveToFile("/tmp/test.json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not supported")

		err = storage.LoadFromFile("/tmp/test.json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not supported")
	})
}

// TestPostgresStorage_Unit юнит-тесты для PostgreSQL хранилища (без реальной БД)
func TestPostgresStorage_Unit(t *testing.T) {
	t.Run("NewPostgresStorage_InvalidMigrationsPath", func(t *testing.T) {
		// Создаем временную БД в памяти для теста
		db, err := sql.Open("postgres", "postgres://test:test@localhost:5432/nonexistent?sslmode=disable")
		if err != nil {
			t.Skip("Cannot create test database connection")
		}
		defer db.Close()

		// Пытаемся создать хранилище с несуществующим путем к миграциям
		_, err = NewPostgresStorage(db, "/nonexistent/path")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to run migrations")
	})
}
