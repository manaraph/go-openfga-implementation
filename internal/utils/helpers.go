package utils

import (
	"context"
	"errors"
	"log"
	"os"
	"strconv"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/manaraph/go-openfga-implementation/pkg/authz"
	"github.com/manaraph/go-openfga-implementation/pkg/db"
	"github.com/manaraph/go-openfga-implementation/pkg/middleware"
	"github.com/openfga/go-sdk/client"
	"go.mongodb.org/mongo-driver/mongo"
)

type AppConfig struct {
	Port        string
	DB          *sqlx.DB
	MongoDB     *mongo.Database
	MongoClient *mongo.Client
	FGA         *client.OpenFgaClient
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(middleware.UserIdKey).(int)
	return strconv.Itoa(userID), ok
}

func InitializeAppConfig() (*AppConfig, error) {
	if err := godotenv.Load(); err != nil {
		return nil, errors.New("No .env file found, reading from env vars")
	}

	port := os.Getenv("PORT")
	if port == "" {
		return nil, errors.New("PORT is required")
	}

	url := os.Getenv("POSTGRES_URL")
	if url == "" {
		return nil, errors.New("POSTGRES_URL is required")
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		return nil, errors.New("MONGO_URI is required")
	}

	mongoDBName := os.Getenv("MONGO_DB")
	if mongoDBName == "" {
		mongoDBName = "files_db"
	}

	fgaUrl := os.Getenv("FGA_URL")
	if fgaUrl == "" {
		return nil, errors.New("FGA_URL is required")
	}

	fgaStoreId := os.Getenv("FGA_STORE_ID")
	if fgaStoreId == "" {
		return nil, errors.New("FGA_STORE_ID is required")
	}

	fgaAuthId := os.Getenv("FGA_AUTH_MODEL_ID")
	if fgaStoreId == "" {
		log.Fatal("FGA_AUTH_MODEL_ID is required")
		return nil, errors.New("FGA_AUTH_MODEL_ID is required")
	}

	// Connect to Postgres DB
	pg, err := db.ConnectPostgres(url)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	// Connect to MongoDB
	mg, err := db.ConnectMongo(mongoURI, mongoDBName)
	if err != nil {
		log.Fatal("failed to connect to mongo:", err)
		return nil, err
	}

	// Initialize OpenFGA client
	fga, err := authz.NewFGAClient(fgaUrl, fgaStoreId, fgaAuthId)
	if err != nil {
		log.Fatal("failed to initialize FGA client:", err)
		return nil, err
	}

	return &AppConfig{
		Port:        port,
		DB:          pg.DB,
		MongoDB:     mg.DB,
		MongoClient: mg.Client,
		FGA:         fga,
	}, nil
}
