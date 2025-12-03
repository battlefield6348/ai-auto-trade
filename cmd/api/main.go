package main

import (
	"log"
	"net/http"
)

func main() {
	srv := newServer()
	addr := ":8080"
	log.Printf("starting HTTP server on %s", addr)
	if err := http.ListenAndServe(addr, srv.routes()); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
