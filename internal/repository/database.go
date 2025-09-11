package repository

import (
	"context"
	"database/sql"
)

// Database интерфейс для работы с базой данных
type Database interface {
	// Ping проверяет соединение с базой данных
	Ping(ctx context.Context) error
	// Close закрывает соединение с базой данных
	Close() error
	// GetConnection возвращает объект соединения с базой данных
	GetConnection() *sql.DB
}

// PostgresDB реализация интерфейса Database для PostgreSQL
type PostgresDB struct {
	DB *sql.DB // Экспортируемое поле для создания из существующего соединения
}

// NewPostgresDB создает новое подключение к PostgreSQL
func NewPostgresDB(dsn string) (*PostgresDB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	// Проверяем соединение
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return &PostgresDB{DB: db}, nil
}

// Ping проверяет соединение с базой данных
func (p *PostgresDB) Ping(ctx context.Context) error {
	return p.DB.PingContext(ctx)
}

// Close закрывает соединение с базой данных
func (p *PostgresDB) Close() error {
	return p.DB.Close()
}

// GetConnection возвращает объект соединения с базой данных
func (p *PostgresDB) GetConnection() *sql.DB {
	return p.DB
}
