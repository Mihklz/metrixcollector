package config

import (
	"flag"
	"time"
)

type AgentConfig struct {
	ServerAddr    string
	PollInterval  time.Duration
	ReportInterval time.Duration
}

func LoadAgentConfig() *AgentConfig {
	var (
		serverAddr    string
		pollSec       int
		reportSec     int
	)

	flag.StringVar(&serverAddr, "a", "localhost:8080", "address of HTTP server")
	flag.IntVar(&pollSec, "p", 2, "poll interval in seconds")
	flag.IntVar(&reportSec, "r", 10, "report interval in seconds")
	flag.Parse()

	return &AgentConfig{
		ServerAddr:     "http://" + serverAddr,
		PollInterval:   time.Duration(pollSec) * time.Second,
		ReportInterval: time.Duration(reportSec) * time.Second,
	}
} 