package main

import (
	"context"
	"example/db"
	"log"
)

func main() {
	db.ConnectDatabase()
	defer db.CloseDatabase()

	ctx := context.Background()

	// Example: Add a new user
	user, err := db.AddUser(ctx, "Johns Doe", "johnss.doe@example.com")
	if err != nil {
		log.Fatal("Failed to add user:", err)
	}
	log.Printf("Added user: %+v", user)

	// Example: Get the user we just created
	retrievedUser, err := db.GetUser(ctx, user.ID)
	if err != nil {
		log.Fatal("Failed to get user:", err)
	}
	log.Printf("Retrieved user: %+v", retrievedUser)

	// Example: Add another user
	user2, err := db.AddUser(ctx, "Janeee Smith", "janeee.smith@example.com")
	if err != nil {
		log.Fatal("Failed to add second user:", err)
	}
	log.Printf("Added second user: %+v", user2)
}
