package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		log.Println("received event")
		w.WriteHeader(http.StatusOK)
	})
	port := os.Getenv("PORT")
	log.Println("notification service running on :", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
