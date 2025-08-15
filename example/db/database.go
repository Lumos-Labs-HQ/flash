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

func CloseDatabase() {
	if DB != nil {
		DB.Close()
		log.Println("Database connection closed")
	}
}

// ========== USERS ==========
func AddUser(ctx context.Context, name, email string) (*graft.User, error) {
	user, err := Queries.CreateUser(ctx, graft.CreateUserParams{Name: name, Email: email})
	if err != nil {
		log.Printf("Error creating user: %v", err)
		return nil, err
	}
	log.Printf("User created: %+v", user)
	return &user, nil
}

func GetUser(ctx context.Context, userID int32) (*graft.User, error) {
	user, err := Queries.GetUser(ctx, userID)
	if err != nil {
		log.Printf("Error getting user with ID %d: %v", userID, err)
		return nil, err
	}
	return &user, nil
}

// ========== CATEGORIES ==========
func AddCategory(ctx context.Context, name string) (*graft.Category, error) {
	category, err := Queries.CreateCategory(ctx, name)
	if err != nil {
		log.Printf("Error creating category: %v", err)
		return nil, err
	}
	log.Printf("Category created: %+v", category)
	return &category, nil
}

// ========== POSTS ==========
func AddPost(ctx context.Context, userID, categoryID int32, title, content string) (*graft.Post, error) {
	post, err := Queries.CreatePost(ctx, graft.CreatePostParams{
		UserID:     userID,
		CategoryID: categoryID,
		Title:      title,
		Content:    content,
	})
	if err != nil {
		log.Printf("Error creating post: %v", err)
		return nil, err
	}
	log.Printf("Post created: %+v", post)
	return &post, nil
}

// ========== COMMENTS ==========
func AddComment(ctx context.Context, postID, userID int32, text string) (*graft.Comment, error) {
	comment, err := Queries.CreateComment(ctx, graft.CreateCommentParams{
		PostID:  postID,
		UserID:  userID,
		Content: text,
	})
	if err != nil {
		log.Printf("Error creating comment: %v", err)
		return nil, err
	}
	log.Printf("Comment added: %+v", comment)
	return &comment, nil
}
