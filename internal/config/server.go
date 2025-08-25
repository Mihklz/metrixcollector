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
}

func LoadServerConfig() *ServerConfig {
	var runAddr string
	var storeInterval int
	var fileStoragePath string
	var restore bool

	// 1. Устанавливаем значения по умолчанию
	flag.StringVar(&runAddr, "a", "localhost:8080", "address and port to run HTTP server")
	flag.IntVar(&storeInterval, "i", 300, "store interval in seconds (0 for synchronous)")
	flag.StringVar(&fileStoragePath, "f", "/tmp/metrics-db.json", "file storage path")
	flag.BoolVar(&restore, "r", true, "restore previously saved values")
	flag.Parse()

	// 2. Проверяем переменные окружения (приоритет выше флагов)
	if envRunAddr := os.Getenv("ADDRESS"); envRunAddr != "" {
		runAddr = envRunAddr
	}

	if envStoreInterval := os.Getenv("STORE_INTERVAL"); envStoreInterval != "" {
		if interval, err := strconv.Atoi(envStoreInterval); err == nil {
			storeInterval = interval
		}
	}

	if envFileStoragePath := os.Getenv("FILE_STORAGE_PATH"); envFileStoragePath != "" {
		fileStoragePath = envFileStoragePath
	}

	if envRestore := os.Getenv("RESTORE"); envRestore != "" {
		if restoreValue, err := strconv.ParseBool(envRestore); err == nil {
			restore = restoreValue
		}
	}

	return &ServerConfig{
		RunAddr:         runAddr,
		StoreInterval:   storeInterval,
		FileStoragePath: fileStoragePath,
		Restore:         restore,
	}
}
