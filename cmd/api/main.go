package main

import (
	"log"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/manaraph/go-openfga-implementation/pkg/db"
	"github.com/manaraph/go-openfga-implementation/pkg/handler"
	"github.com/manaraph/go-openfga-implementation/pkg/server"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from env vars")
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
	log.Println("Connected to MongoDB")

	r := chi.NewRouter()
	h := handler.New(pg.DB, mg.Database)
	h.RegisterRoutes(r)

	srv := server.New(":8080", r)
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
}
