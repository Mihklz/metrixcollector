package config

import (
	"flag"
	"os"
	"strconv"
)

type ServerConfig struct {
	RunAddr         string
	StoreInterval   int    // интервал сохранения в секундах
	FileStoragePath string // путь к файлу для сохранения метрик
	Restore         bool   // загружать ли метрики при старте
	DatabaseDSN     string // строка подключения к базе данных
	Key             string // ключ для подписи данных
	AuditFile       string // путь к файлу для логов аудита
	AuditURL        string // URL для отправки логов аудита
}

func LoadServerConfig() *ServerConfig {
	var runAddr string
	var storeInterval int
	var fileStoragePath string
	var restore bool
	var databaseDSN string
	var key string
	var auditFile string
	var auditURL string

	// 1. Устанавливаем значения по умолчанию
	flag.StringVar(&runAddr, "a", "localhost:8080", "address and port to run HTTP server")
	flag.IntVar(&storeInterval, "i", 300, "store interval in seconds (0 for synchronous)")
	flag.StringVar(&fileStoragePath, "f", "/tmp/metrics-db.json", "file storage path")
	flag.BoolVar(&restore, "r", true, "restore previously saved values")
	flag.StringVar(&databaseDSN, "d", "", "database connection string")
	flag.StringVar(&key, "k", "", "key for signing data")
	flag.StringVar(&auditFile, "audit-file", "", "audit log file path")
	flag.StringVar(&auditURL, "audit-url", "", "audit log URL")
	flag.Parse()

	// 2. Проверяем переменные окружения (приоритет выше флагов)
	if envRunAddr, ok := os.LookupEnv("ADDRESS"); ok {
		runAddr = envRunAddr
	}

	if envStoreInterval, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		if interval, err := strconv.Atoi(envStoreInterval); err == nil {
			storeInterval = interval
		}
	}

	if envFileStoragePath, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		fileStoragePath = envFileStoragePath
	}

	if envRestore, ok := os.LookupEnv("RESTORE"); ok {
		if restoreValue, err := strconv.ParseBool(envRestore); err == nil {
			restore = restoreValue
		}
	}

	if envDatabaseDSN, ok := os.LookupEnv("DATABASE_DSN"); ok {
		databaseDSN = envDatabaseDSN
	}

	if envKey, ok := os.LookupEnv("KEY"); ok {
		key = envKey
	}

	if envAuditFile, ok := os.LookupEnv("AUDIT_FILE"); ok {
		auditFile = envAuditFile
	}

	if envAuditURL, ok := os.LookupEnv("AUDIT_URL"); ok {
		auditURL = envAuditURL
	}

	return &ServerConfig{
		RunAddr:         runAddr,
		StoreInterval:   storeInterval,
		FileStoragePath: fileStoragePath,
		Restore:         restore,
		DatabaseDSN:     databaseDSN,
		Key:             key,
		AuditFile:       auditFile,
		AuditURL:        auditURL,
	}
}
