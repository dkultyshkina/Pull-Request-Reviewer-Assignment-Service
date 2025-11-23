package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"fmt"
	"time"
	"context"

	_ "github.com/lib/pq" 

	"service/internal/handler"
)

func connectToDB() (*sql.DB, error) {
	dbHost := "db"
	dbPort := "5432"
	dbUser := "reviewer_user"
	dbPassword := "password"
	dbName := "reviewer"
	dbSSL := "disable"
	db, err := sql.Open("postgres", fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSL,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	return db, nil
}

func getPort() string {
	if port := os.Getenv("PORT"); port != "" {
		return port
	}
	return "8080"
}

func setupRoutes(h *handlers.Handlers) {
	if h == nil {
		log.Fatal("Handlers is nil in setup")
	}
	http.HandleFunc("/team/add", h.AddTeam)
	http.HandleFunc("/team/get", h.GetTeam)
	http.HandleFunc("/users/setIsActive", h.SetUserActive)
	http.HandleFunc("/users/getReview", h.GetUserReviewPRs)
	http.HandleFunc("/pullRequest/create", h.CreatePR)
	http.HandleFunc("/pullRequest/merge", h.MergePR)
	http.HandleFunc("/pullRequest/reassign", h.ReassignReviewer)
	http.HandleFunc("/stats", h.GetStats)
	http.HandleFunc("/health", h.Health)
}