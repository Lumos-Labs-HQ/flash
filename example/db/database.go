package db

import (
	"context"
	"log"
	"os"

	graft "example/graft_gen"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var DB *pgxpool.Pool
var Queries *graft.Queries

func ConnectDatabase() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	var err error
	DB, err = pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	if err = DB.Ping(context.Background()); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	Queries = graft.New(DB)
	log.Println("Database connected successfully")
}

func AddUser(ctx context.Context, name, email string) (*graft.User, error) {
	params := graft.CreateUserParams{
		Name:  name,
		Email: email,
	}

	user, err := Queries.CreateUser(ctx, params)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		return nil, err
	}

	log.Printf("User created successfully: ID=%d, Name=%s, Email=%s", user.ID, user.Name, user.Email)
	return &user, nil
}

func GetUser(ctx context.Context, userID int32) (*graft.User, error) {
	user, err := Queries.GetUser(ctx, userID)
	if err != nil {
		log.Printf("Error getting user with ID %d: %v", userID, err)
		return nil, err
	}

	log.Printf("User retrieved: ID=%d, Name=%s, Email=%s", user.ID, user.Name, user.Email)
	return &user, nil
}

func CloseDatabase() {
	if DB != nil {
		DB.Close()
		log.Println("Database connection closed")
	}
}
