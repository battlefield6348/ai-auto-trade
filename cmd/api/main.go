package main

import (
	"log"
	"net/http"

	httpapi "ai-auto-trade/internal/interface/http"
)

func main() {
	router := httpapi.NewRouter()

	addr := ":8080"
	log.Printf("starting HTTP server on %s", addr)

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
