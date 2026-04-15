// cmd/server/main.go

package main

import (
	"log"
	"net/http"
	"resistance/internal/api"
	"resistance/internal/config"
)

func main() {
	cfg := config.LoadConfig()
	log.Print("Loaded config")
	
	server := api.NewServer(cfg)
	log.Print("Created server")
	
	router := api.NewRouter(server)
	log.Print("Initialized router")

	log.Printf("Starting server on port: %s\n", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+server.Config.Port, router))
}
