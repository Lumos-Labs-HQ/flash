package db

import (
	"database/sql"
	"log"
	"os"

	graft_gen "example/graft_gen"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

var DB *sql.DB
var Queries *graft_gen.Queries

func ConnectDatabase() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	config, err := pgx.ParseConfig(dbURL)
	if err != nil {
		log.Fatal("Failed to parse connection URL:", err)
	}

	config.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	DB = stdlib.OpenDB(*config)

	if err = DB.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	Queries = graft_gen.New(DB)
	log.Println("Database connected successfully")
}

func CloseDatabase() {
	if DB != nil {
		DB.Close()
		log.Println("Database connection closed")
	}
}

// ========== USERS ==========
func AddUser(name, email string) (*graft_gen.CreateuserRow, error) {
	user, err := Queries.Createuser(name, email)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		return nil, err
	}
	log.Printf("User created: %+v", user)
	return &user, nil
}

func GetUser(userID int64) (*graft_gen.GetuserRow, error) {
	user, err := Queries.Getuser(userID)
	if err != nil {
		log.Printf("Error getting user with ID %d: %v", userID, err)
		return nil, err
	}
	return &user, nil
}

// ========== CATEGORIES ==========
func AddCategory(name string) (*graft_gen.CreatecategoryRow, error) {
	category, err := Queries.Createcategory(name)
	if err != nil {
		log.Printf("Error creating category: %v", err)
		return nil, err
	}
	log.Printf("Category created: %+v", category)
	return &category, nil
}

func GetUserByEmail(email string) (*graft_gen.GetuserbyemailRow, error) {
	user, err := Queries.Getuserbyemail(email)
	if err != nil {
		log.Printf("Error getting user with email %s: %v", email, err)
		return nil, err
	}
	return &user, nil
}

// ========== POSTS ==========
func AddPost(userID, categoryID int64, title, content string) (*graft_gen.CreatepostRow, error) {
	post, err := Queries.Createpost(graft_gen.CreatepostParams{
		UserId:     userID,
		CategoryId: categoryID,
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
func AddComment(postID, userID int64, text string) (*graft_gen.CreatecommentRow, error) {
	comment, err := Queries.Createcomment(postID, userID, text)
	if err != nil {
		log.Printf("Error creating comment: %v", err)
		return nil, err
	}
	log.Printf("Comment added: %+v", comment)
	return &comment, nil
}
