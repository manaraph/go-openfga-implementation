package main

import (
	"log"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/manaraph/go-openfga-implementation/pkg/authz"
	"github.com/manaraph/go-openfga-implementation/pkg/db"
	"github.com/manaraph/go-openfga-implementation/pkg/handler"
	"github.com/manaraph/go-openfga-implementation/pkg/server"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from env vars")
	}

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT is required")
	}

	url := os.Getenv("POSTGRES_URL")
	if url == "" {
		log.Fatal("POSTGRES_URL is required")
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Fatal("MONGO_URI is required")
	}

	mongoDBName := os.Getenv("MONGO_DB")
	if mongoDBName == "" {
		mongoDBName = "files_db"
	}

	fgaUrl := os.Getenv("FGA_URL")
	if fgaUrl == "" {
		log.Fatal("FGA_URL is required")
	}

	fgaStoreId := os.Getenv("FGA_STORE_ID")
	if fgaStoreId == "" {
		log.Fatal("FGA_STORE_ID is required")
	}

	fgaAuthId := os.Getenv("FGA_AUTH_MODEL_ID")
	if fgaStoreId == "" {
		log.Fatal("FGA_AUTH_MODEL_ID is required")
	}

	// Connect to Postgres DB
	pg, err := db.ConnectPostgres(url)
	if err != nil {
		log.Fatal(err)
	}

	// Connect to MongoDB
	mg, err := db.ConnectMongo(mongoURI, mongoDBName)
	if err != nil {
		log.Fatal("failed to connect to mongo:", err)
	}

	// Initialize OpenFGA client
	fga, err := authz.NewFGAClient(fgaUrl, fgaStoreId, fgaAuthId)
	if err != nil {
		log.Fatal("failed to initialize FGA client:", err)
	}

	r := chi.NewRouter()
	h := handler.New(pg.DB, mg.DB, fga)
	h.RegisterRoutes(r)

	srv := server.New(":"+port, r)
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
}
