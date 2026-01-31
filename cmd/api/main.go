package main

import (
	"log"

	"github.com/go-chi/chi/v5"
	"github.com/manaraph/go-openfga-implementation/internal/handler"
	"github.com/manaraph/go-openfga-implementation/internal/server"
)

func main() {
	r := chi.NewRouter()
	h := handler.New()
	h.RegisterRoutes(r)

	srv := server.New(":8080", r)
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
}
