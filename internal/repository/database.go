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
	db *sql.DB
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

	return &PostgresDB{db: db}, nil
}

// Ping проверяет соединение с базой данных
func (p *PostgresDB) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

// Close закрывает соединение с базой данных
func (p *PostgresDB) Close() error {
	return p.db.Close()
}

// GetConnection возвращает объект соединения с базой данных
func (p *PostgresDB) GetConnection() *sql.DB {
	return p.db
}
