package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"

	"github.com/Mihklz/metrixcollector/internal/logger"
)

// PostgresStorage реализует интерфейс Storage для PostgreSQL
type PostgresStorage struct {
	db *sql.DB
}

// NewPostgresStorage создает новое PostgreSQL хранилище
func NewPostgresStorage(db *sql.DB, migrationsPath string) (*PostgresStorage, error) {
	storage := &PostgresStorage{db: db}
	
	// Применяем миграции
	if err := storage.runMigrations(migrationsPath); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}
	
	return storage, nil
}

// runMigrations применяет миграции к базе данных
func (ps *PostgresStorage) runMigrations(migrationsPath string) error {
	driver, err := postgres.WithInstance(ps.db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	// Применяем миграции
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	logger.Log.Info("Database migrations applied successfully")
	return nil
}

// Update обновляет или создает метрику
func (ps *PostgresStorage) Update(metricType, name, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch metricType {
	case "gauge":
		return ps.updateGauge(ctx, name, value)
	case "counter":
		return ps.updateCounter(ctx, name, value)
	default:
		return fmt.Errorf("unsupported metric type: %s", metricType)
	}
}

// updateGauge обновляет gauge метрику
func (ps *PostgresStorage) updateGauge(ctx context.Context, name, value string) error {
	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fmt.Errorf("invalid gauge value: %w", err)
	}

	query := `
		INSERT INTO metrics (name, type, value, updated_at) 
		VALUES ($1, 'gauge', $2, CURRENT_TIMESTAMP)
		ON CONFLICT (name, type) 
		DO UPDATE SET value = EXCLUDED.value, updated_at = CURRENT_TIMESTAMP`

	_, err = ps.db.ExecContext(ctx, query, name, floatValue)
	if err != nil {
		return fmt.Errorf("failed to update gauge metric: %w", err)
	}

	logger.Log.Debug("Gauge metric updated",
		zap.String("name", name),
		zap.Float64("value", floatValue),
	)
	return nil
}

// updateCounter обновляет counter метрику (добавляет к существующему значению)
func (ps *PostgresStorage) updateCounter(ctx context.Context, name, value string) error {
	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid counter value: %w", err)
	}

	query := `
		INSERT INTO metrics (name, type, delta, updated_at) 
		VALUES ($1, 'counter', $2, CURRENT_TIMESTAMP)
		ON CONFLICT (name, type) 
		DO UPDATE SET delta = metrics.delta + EXCLUDED.delta, updated_at = CURRENT_TIMESTAMP`

	_, err = ps.db.ExecContext(ctx, query, name, intValue)
	if err != nil {
		return fmt.Errorf("failed to update counter metric: %w", err)
	}

	logger.Log.Debug("Counter metric updated",
		zap.String("name", name),
		zap.Int64("delta", intValue),
	)
	return nil
}

// GetGauge возвращает gauge метрику
func (ps *PostgresStorage) GetGauge(name string) (Gauge, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var value float64
	query := `SELECT value FROM metrics WHERE name = $1 AND type = 'gauge'`
	
	err := ps.db.QueryRowContext(ctx, query, name).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, false
		}
		logger.Log.Error("Failed to get gauge metric", zap.Error(err), zap.String("name", name))
		return 0, false
	}

	return Gauge(value), true
}

// GetCounter возвращает counter метрику
func (ps *PostgresStorage) GetCounter(name string) (Counter, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var delta int64
	query := `SELECT delta FROM metrics WHERE name = $1 AND type = 'counter'`
	
	err := ps.db.QueryRowContext(ctx, query, name).Scan(&delta)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, false
		}
		logger.Log.Error("Failed to get counter metric", zap.Error(err), zap.String("name", name))
		return 0, false
	}

	return Counter(delta), true
}

// GetAllGauges возвращает все gauge метрики
func (ps *PostgresStorage) GetAllGauges() map[string]Gauge {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	gauges := make(map[string]Gauge)
	query := `SELECT name, value FROM metrics WHERE type = 'gauge'`
	
	rows, err := ps.db.QueryContext(ctx, query)
	if err != nil {
		logger.Log.Error("Failed to get all gauge metrics", zap.Error(err))
		return gauges
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var value float64
		if err := rows.Scan(&name, &value); err != nil {
			logger.Log.Error("Failed to scan gauge metric", zap.Error(err))
			continue
		}
		gauges[name] = Gauge(value)
	}

	if err := rows.Err(); err != nil {
		logger.Log.Error("Error iterating gauge metrics", zap.Error(err))
	}

	return gauges
}

// GetAllCounters возвращает все counter метрики
func (ps *PostgresStorage) GetAllCounters() map[string]Counter {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	counters := make(map[string]Counter)
	query := `SELECT name, delta FROM metrics WHERE type = 'counter'`
	
	rows, err := ps.db.QueryContext(ctx, query)
	if err != nil {
		logger.Log.Error("Failed to get all counter metrics", zap.Error(err))
		return counters
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var delta int64
		if err := rows.Scan(&name, &delta); err != nil {
			logger.Log.Error("Failed to scan counter metric", zap.Error(err))
			continue
		}
		counters[name] = Counter(delta)
	}

	if err := rows.Err(); err != nil {
		logger.Log.Error("Error iterating counter metrics", zap.Error(err))
	}

	return counters
}

// SaveToFile не поддерживается для PostgreSQL хранилища
func (ps *PostgresStorage) SaveToFile(filename string) error {
	return fmt.Errorf("SaveToFile not supported for PostgreSQL storage")
}

// LoadFromFile не поддерживается для PostgreSQL хранилища
func (ps *PostgresStorage) LoadFromFile(filename string) error {
	return fmt.Errorf("LoadFromFile not supported for PostgreSQL storage")
}

// Close закрывает соединение с базой данных
func (ps *PostgresStorage) Close() error {
	return ps.db.Close()
}

// GetConnection возвращает объект соединения с базой данных для ping handler
func (ps *PostgresStorage) GetConnection() *sql.DB {
	return ps.db
}
