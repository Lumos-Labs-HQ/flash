package main

import (
	"context"
	"fmt"
	"time"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/bson"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	url := "mongodb+srv://sdk-snapshot:sdk-snapshot@sdk-snapshot.0mmyw.mongodb.net/test"
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(url))
	if err != nil {
		fmt.Printf("Connect error: %v\n", err)
		return
	}
	defer client.Disconnect(ctx)
	
	if err := client.Ping(ctx, nil); err != nil {
		fmt.Printf("Ping error: %v\n", err)
		return
	}
	
	fmt.Println("✅ Connected successfully!")
	
	db := client.Database("test")
	collections, err := db.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		fmt.Printf("List collections error: %v\n", err)
		return
	}
	
	fmt.Printf("Collections found: %d\n", len(collections))
	for _, name := range collections {
		fmt.Printf("  - %s\n", name)
	}
}
