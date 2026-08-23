package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	server := newStoreServer(newInfraiClient(key))
	log.Printf("store session service listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, server.routes()))
}
