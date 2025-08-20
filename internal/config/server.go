package config

import (
	"flag"
	"os"
)

type ServerConfig struct {
	RunAddr string
}

func LoadServerConfig() *ServerConfig {
	var runAddr string

	// 1. Устанавливаем значение по умолчанию
	flag.StringVar(&runAddr, "a", "localhost:8080", "address and port to run HTTP server")
	flag.Parse()

	// 2. Проверяем переменную окружения (приоритет выше флагов)
	if envRunAddr := os.Getenv("ADDRESS"); envRunAddr != "" {
		runAddr = envRunAddr
	}

	return &ServerConfig{
		RunAddr: runAddr,
	}
}
