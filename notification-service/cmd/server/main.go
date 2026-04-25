package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kernelshard/url-shortner-platform/notification-service/internal/handler"
	"github.com/kernelshard/url-shortner-platform/notification-service/internal/repository"
)

func main() {
	ctx := context.Background()
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL environment variable not set")
	}
	dbPool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer dbPool.Close()

	repo := repository.NewPostgresProcessedEventRepository(dbPool)
	emailSVC := handler.NewEmailService()

	h := handler.NewHttpHandler(repo, emailSVC)

	http.HandleFunc("/events", h.HandleEvents)

	port := os.Getenv("PORT")
	log.Println("notification service running on :", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
