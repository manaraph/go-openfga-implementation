package main

import (
	"log"

	"github.com/go-chi/chi/v5"
	"github.com/manaraph/go-openfga-implementation/internal/utils"
	"github.com/manaraph/go-openfga-implementation/pkg/handler"
	"github.com/manaraph/go-openfga-implementation/pkg/server"
)

func main() {
	config, err := utils.InitializeAppConfig()
	if err != nil {
		log.Fatal("failed to initialize FGA client:", err)
	}

	r := chi.NewRouter()
	h := handler.New(config.DB, config.MongoDB, config.FGA)
	h.RegisterRoutes(r)

	srv := server.New(":"+config.Port, r)
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
}
