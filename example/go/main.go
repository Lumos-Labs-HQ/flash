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

	// Insert Users
	users := []struct {
		Name  string
		Email string
	}{
		{"Alice Johnson", "alice.johnson@example.com"},
		{"Bob Smith", "bob.smith@example.com"},
		{"Charlie Brown", "charlie.brown@example.com"},
		{"Diana Prince", "diana.prince@example.com"},
		{"Ethan Hunt", "ethan.hunt@example.com"},
	}

	var userIDs []int32
	for _, u := range users {
		user, err := db.AddUser(ctx, u.Name, u.Email)
		if err == nil {
			userIDs = append(userIDs, user.ID)
		}
	}

	// Insert Categories
	categories := []string{"Technology", "Lifestyle", "Travel", "Food", "Sports"}
	var categoryIDs []int32
	for _, name := range categories {
		category, err := db.AddCategory(ctx, name)
		if err == nil {
			categoryIDs = append(categoryIDs, category.ID)
		}
	}

	// Insert Posts
	posts := []struct {
		UserID     int32
		CategoryID int32
		Title      string
		Content    string
	}{
		{userIDs[0], categoryIDs[0], "The Future of AI", "Artificial Intelligence is transforming industries at a rapid pace."},
		{userIDs[1], categoryIDs[1], "Minimalist Living", "Living with less can bring more happiness."},
		{userIDs[2], categoryIDs[2], "Backpacking Through Europe", "Tips for an affordable trip across Europe."},
		{userIDs[3], categoryIDs[3], "10 Delicious Pasta Recipes", "From classic Italian to modern fusion."},
		{userIDs[4], categoryIDs[4], "Top 10 Football Players in 2025", "A list of the best players in the world right now."},
	}

	var postIDs []int32
	for _, p := range posts {
		post, err := db.AddPost(ctx, p.UserID, p.CategoryID, p.Title, p.Content)
		if err == nil {
			postIDs = append(postIDs, post.ID)
		}
	}

	// Insert Comments
	comments := []struct {
		PostID int32
		UserID int32
		Text   string
	}{
		{postIDs[0], userIDs[1], "Great insights! AI will rule the world."},
		{postIDs[0], userIDs[2], "Very informative, thanks for sharing."},
		{postIDs[1], userIDs[0], "I love minimalist living too."},
		{postIDs[2], userIDs[3], "Europe is beautiful in spring!"},
		{postIDs[3], userIDs[4], "I tried recipe #3, amazing!"},
		{postIDs[4], userIDs[0], "Messi is still the GOAT!"},
	}

	for _, c := range comments {
		_, err := db.AddComment(ctx, c.PostID, c.UserID, c.Text)
		if err != nil {
			log.Printf("Failed to insert comment: %s", c.Text)
		}
	}

	log.Println("All data inserted successfully!")
}
