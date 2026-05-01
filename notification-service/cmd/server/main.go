package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kernelshard/url-shortner-platform/notification-service/internal/handler"
	"github.com/kernelshard/url-shortner-platform/notification-service/internal/repository"
	"github.com/kernelshard/url-shortner-platform/notification-service/internal/service"
	"github.com/kernelshard/url-shortner-platform/notification-service/internal/worker"
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
	eventSvc := service.NewEventService(repo)
	emailSvc := service.NewEmailService()

	emailRepo := repository.NewPostgresEmailDeliveryRepository(dbPool)

	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go worker.RunWorker(workerCtx, emailRepo, emailSvc)

	h := handler.NewHttpHandler(eventSvc)

	http.HandleFunc("/events", h.HandleEvents)

	port := os.Getenv("PORT")
	log.Println("notification service running on :", port)
	// log.Fatal(http.ListenAndServe(":"+port, nil))
	//
	go func() {
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	<-quit
	cancel()
	time.Sleep(5 * time.Second)
}
