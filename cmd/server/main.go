package main

import (
	"github.com/Mihklz/metrixcollector/internal/handler"
	"github.com/Mihklz/metrixcollector/internal/repository"
	"log"
	"net/http"
)

func main() {
	storage := repository.NewMemStorage()

	mux := http.NewServeMux()
	mux.HandleFunc("/update/", handler.NewUpdateHandler(storage))

	log.Println("Starting server at :8080")
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
