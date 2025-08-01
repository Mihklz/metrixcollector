package config

import (
	"flag"
)

type ServerConfig struct {
	RunAddr string
}

func LoadServerConfig() *ServerConfig {
	var runAddr string

	flag.StringVar(&runAddr, "a", "localhost:8080", "address and port to run HTTP server")
	flag.Parse()

	return &ServerConfig{
		RunAddr: runAddr,
	}
} 