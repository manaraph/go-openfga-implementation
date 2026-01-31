package db

import (
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var DB *sqlx.DB

func Init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, ensure you have created or copied the file from the .env.example")
	}

	url := os.Getenv("POSTGRES_URL")
	if url == "" {
		log.Fatal("POSTGRES_URL is required")
	}

	db, err := sqlx.Connect("postgres", url)
	if err != nil {
		log.Fatal("postgres DB: ", err)
	}

	DB = db

	schema := `
	CREATE TABLE IF NOT EXISTS users(
		id SERIAL PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);`

	DB.MustExec(schema)

	log.Println("Connected to Postgres DB")
}
