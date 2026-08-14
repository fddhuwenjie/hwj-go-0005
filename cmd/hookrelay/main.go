package main

import (
	"log"
	"net/http"
	"os"

	"github.com/fddhuwenjie/hwj-go-0005/relay"
)

func main() {
	addr := os.Getenv("HOOK_RELAY_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	server := relay.NewServer(relay.NewStore())
	log.Printf("hook relay listening on %s", addr)
	if err := http.ListenAndServe(addr, server); err != nil {
		log.Fatal(err)
	}
}
