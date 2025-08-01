package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/Mihklz/metrixcollector/internal/handler"
	"github.com/Mihklz/metrixcollector/internal/repository"
	"github.com/Mihklz/metrixcollector/internal/config"
)

func main() {
	cfg := config.LoadServerConfig()

	storage := repository.NewMemStorage()

	// Инициализируем chi-роутер
	r := chi.NewRouter()

	// POST /update/{type}/{name}/{value}
	r.Post("/update/{type}/{name}/{value}", handler.NewUpdateHandler(storage))

	// GET /value/{type}/{name}
	r.Get("/value/{type}/{name}", handler.NewValueHandler(storage))

	// GET /
	r.Get("/", handler.NewRootHandler(storage))
	

	// Запускаем сервер
	log.Printf("Starting server at %s", cfg.RunAddr)
	if err := http.ListenAndServe(cfg.RunAddr, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
