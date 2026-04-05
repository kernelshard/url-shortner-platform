package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kernelshard/url-shortner-platform/internal/cache"
	"github.com/kernelshard/url-shortner-platform/internal/event"

	httpTransport "github.com/kernelshard/url-shortner-platform/internal/handler/http"
	"github.com/kernelshard/url-shortner-platform/internal/repository"
	"github.com/kernelshard/url-shortner-platform/internal/service"
)

func main() {
	ctx := context.Background()

	// conn, err := pgx.Connect(ctx, "postgres://postgres:postgres@localhost:5432/shortner?sslmode=disable")
	conn, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))

	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer conn.Close()

	// dependency injection
	repo := repository.NewPostgresLinkRepository(conn)
	cache := cache.NewRedisCache(os.Getenv("REDIS_URL"))

	pub := event.NewHTTPPublisher()
	
	svc := service.NewLinkService(repo, cache, pub)

	h := httpTransport.NewHandler(svc)

	mux := http.NewServeMux()

	// Define the HTTP endpoints and associate them with handler functions
	mux.HandleFunc("POST /urls", h.CreateShortURL)
	mux.HandleFunc("GET /r/{code}", h.Redirect)

	port := ":8080"
	log.Println("Starting server on port", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}

}
