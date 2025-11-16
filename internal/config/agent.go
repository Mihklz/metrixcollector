package config

import (
	"flag"
	"log"
	"os"
	"strconv"
	"time"
)

type AgentConfig struct {
	ServerAddr     string
	PollInterval   time.Duration
	ReportInterval time.Duration
	Key            string // ключ для подписи данных
	RateLimit      int    // ограничение на число одновременных исходящих запросов
}

func LoadAgentConfig() *AgentConfig {
	var (
		serverAddr string
		pollSec    int
		reportSec  int
		key        string
		rateLimit  int
	)

	// 1. Устанавливаем значения по умолчанию через флаги
	flag.StringVar(&serverAddr, "a", "localhost:8080", "address of HTTP server")
	flag.IntVar(&pollSec, "p", 2, "poll interval in seconds")
	flag.IntVar(&reportSec, "r", 10, "report interval in seconds")
	flag.StringVar(&key, "k", "", "key for signing data")
	flag.IntVar(&rateLimit, "l", 10, "rate limit for concurrent requests")
	flag.Parse()

	// 2. Проверяем переменные окружения (приоритет выше флагов)

	// ADDRESS - адрес сервера
	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		serverAddr = envAddr
	}

	// POLL_INTERVAL - интервал сбора метрик в секундах
	if envPollSec := os.Getenv("POLL_INTERVAL"); envPollSec != "" {
		if parsedPoll, err := strconv.Atoi(envPollSec); err == nil {
			pollSec = parsedPoll
		} else {
			log.Printf("Invalid POLL_INTERVAL value: %s, using default: %d", envPollSec, pollSec)
		}
	}

	// REPORT_INTERVAL - интервал отправки метрик в секундах
	if envReportSec := os.Getenv("REPORT_INTERVAL"); envReportSec != "" {
		if parsedReport, err := strconv.Atoi(envReportSec); err == nil {
			reportSec = parsedReport
		} else {
			log.Printf("Invalid REPORT_INTERVAL value: %s, using default: %d", envReportSec, reportSec)
		}
	}

	// KEY - ключ для подписи данных
	if envKey := os.Getenv("KEY"); envKey != "" {
		key = envKey
	}

	// RATE_LIMIT - ограничение на количество одновременных запросов
	if envRate := os.Getenv("RATE_LIMIT"); envRate != "" {
		if v, err := strconv.Atoi(envRate); err == nil {
			rateLimit = v
		} else {
			log.Printf("Invalid RATE_LIMIT value: %s, using default: %d", envRate, rateLimit)
		}
	}

	return &AgentConfig{
		ServerAddr:     "http://" + serverAddr,
		PollInterval:   time.Duration(pollSec) * time.Second,
		ReportInterval: time.Duration(reportSec) * time.Second,
		Key:            key,
		RateLimit:      rateLimit,
	}
}
