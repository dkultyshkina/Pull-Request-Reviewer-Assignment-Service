package main

import (
	"log"
	"net/http"

	_ "github.com/lib/pq" 

	"service/internal/service"
	"service/internal/handler"
	"service/internal/repository"
)

func main() {
	db, err := connectToDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()
	repo := repository.NewRepository(db)
	if repo == nil {
		log.Fatal("Repository is nil")
	}
	svc := service.NewService(repo)
	if svc == nil {
		log.Fatal("Service is nil")
	}
	handlers := handlers.NewHandlers(svc)
	if handlers == nil {
		log.Fatal("Handlers is nil")
	}
	setupRoutes(handlers)
	port := getPort()
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
