package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/Mihklz/metrixcollector/internal/handler"
	"github.com/Mihklz/metrixcollector/internal/repository"
)

func main() {
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
	log.Println("Starting server at :8080")
	err := http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
