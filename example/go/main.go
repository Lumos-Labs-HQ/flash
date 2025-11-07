package main

import (
	"example/db"
	"example/flash_gen"
	"log"
)

func main() {
	db.ConnectDatabase()
	defer db.CloseDatabase()

	users := []struct {
		Name  string
		Email string
	}{
		{"Alice Johnson", "alice.j@techcorp.com"},
		{"Bob Smith", "bob.smith@devmail.io"},
		{"Charlie Brown", "charlie.b@codebase.dev"},
		{"Diana Prince", "diana.p@wondertech.net"},
		{"Ethan Hunt", "ethan.h@mission.org"},
		{"Fiona Green", "fiona.g@ecotech.com"},
		{"George Wilson", "george.w@datastream.io"},
		{"Hannah Lee", "hannah.l@cloudnine.dev"},
		{"Ivan Petrov", "ivan.p@rustech.ru"},
		{"Julia Martinez", "julia.m@spanishdev.es"},
	}

	var userIDs []int64
	for _, u := range users {
		user, err := db.AddUser(u.Name, u.Email)
		if err == nil {
			userIDs = append(userIDs, user.Id)
		}
	}

	categories := []string{"Technology", "Lifestyle", "Travel", "Food", "Sports", "Science", "Business", "Entertainment"}
	var categoryIDs []int64
	for _, name := range categories {
		category, err := db.AddCategory(name)
		if err == nil {
			categoryIDs = append(categoryIDs, category.Id)
		} else {
			log.Printf("Error creating category: %s", err)
		}
	}

	if len(categoryIDs) == 0 {
		log.Fatal("No categories available. Please clear the database and try again.")
	}

	getCategoryID := func(index int) int64 {
		return categoryIDs[index%len(categoryIDs)]
	}

	getUserID := func(index int) int64 {
		return userIDs[index%len(userIDs)]
	}

	posts := []struct {
		UserID     int64
		CategoryID int64
		Title      string
		Content    string
	}{
		{getUserID(0), getCategoryID(0), "The Future of AI in 2025", "Artificial Intelligence is transforming industries at an unprecedented pace. From healthcare to finance, AI is revolutionizing how we work."},
		{getUserID(1), getCategoryID(1), "Minimalist Living: Less is More", "Discover how living with less can bring more happiness, freedom, and mental clarity to your daily life."},
		{getUserID(2), getCategoryID(2), "Backpacking Through Europe on a Budget", "My journey across 15 European countries in 3 months. Tips for affordable travel and unforgettable experiences."},
		{getUserID(3), getCategoryID(3), "10 Delicious Pasta Recipes You Must Try", "From classic Italian carbonara to modern fusion dishes, these pasta recipes will delight your taste buds."},
		{getUserID(4), getCategoryID(4), "Top Football Players in 2025", "A comprehensive analysis of the world's best football players and their incredible performances this season."},
		{getUserID(5), getCategoryID(5), "Quantum Computing Breakthrough", "Scientists achieve quantum supremacy with a 1000-qubit processor, opening new possibilities for computation."},
		{getUserID(6), getCategoryID(6), "Startup Culture in Silicon Valley", "Inside look at how tech startups are reshaping business practices and company culture in the Bay Area."},
		{getUserID(7), getCategoryID(7), "Best Movies of 2025 So Far", "A curated list of must-watch films that have captivated audiences worldwide this year."},
		{getUserID(8), getCategoryID(0), "Cybersecurity Best Practices", "Essential security measures every developer should implement to protect applications and user data."},
		{getUserID(9), getCategoryID(2), "Hidden Gems in South America", "Explore lesser-known destinations in South America that offer breathtaking landscapes and rich culture."},
		{getUserID(0), getCategoryID(5), "Climate Change Solutions", "Innovative technologies and strategies being developed to combat global warming and environmental degradation."},
		{getUserID(1), getCategoryID(4), "Marathon Training Guide", "Complete 16-week training plan to prepare for your first marathon, with nutrition and recovery tips."},
	}

	var postIDs []int64
	for _, p := range posts {
		post, err := db.AddPost(p.UserID, p.CategoryID, p.Title, p.Content)
		if err == nil {
			postIDs = append(postIDs, post.Id)
		} else {
			log.Printf("Failed to insert post: %s", p.Title)
		}
	}

	if len(postIDs) == 0 {
		log.Fatal("No posts available. Please clear the database and try again.")
	}

	getPostID := func(index int) int64 {
		return postIDs[index%len(postIDs)]
	}

	comments := []struct {
		PostID int64
		UserID int64
		Text   string
	}{
		{getPostID(0), getUserID(1), "Great article! AI is indeed reshaping our future."},
		{getPostID(0), getUserID(2), "I'm concerned about the ethical implications though."},
		{getPostID(0), getUserID(5), "The quantum computing angle is particularly interesting!"},

		{getPostID(1), getUserID(0), "I've been trying this for a year now. Life-changing!"},
		{getPostID(1), getUserID(3), "Any tips on where to start?"},
		{getPostID(1), getUserID(7), "Less clutter = less stress. So true!"},

		{getPostID(2), getUserID(4), "Which country was your favorite?"},
		{getPostID(2), getUserID(6), "The budget tips are very helpful, thank you!"},
		{getPostID(2), getUserID(8), "I did something similar last year. Amazing experience!"},
		{getPostID(2), getUserID(9), "Adding this to my bucket list!"},

		{getPostID(3), getUserID(2), "The carbonara recipe looks delicious!"},
		{getPostID(3), getUserID(5), "Can't wait to try these this weekend!"},
		{getPostID(3), getUserID(7), "As an Italian, I approve of these recipes 👌"},

		{getPostID(4), getUserID(0), "What about Messi? He should be on this list."},
		{getPostID(4), getUserID(3), "The analysis is spot on!"},
		{getPostID(4), getUserID(6), "Great stats and insights!"},

		{getPostID(5), getUserID(1), "This is revolutionary for cryptography!"},
		{getPostID(5), getUserID(4), "Can't wait to see practical applications."},
		{getPostID(5), getUserID(8), "The science behind this is mind-blowing."},

		{getPostID(6), getUserID(2), "Having worked in SV, this is incredibly accurate."},
		{getPostID(6), getUserID(7), "The work-life balance aspect is concerning."},
		{getPostID(6), getUserID(9), "Innovative culture but intense pressure."},

		{getPostID(7), getUserID(0), "Finally! Been waiting for this list!"},
		{getPostID(7), getUserID(3), "Half of these are on my watchlist already."},
		{getPostID(7), getUserID(5), "Great picks! You have excellent taste!"},

		{getPostID(8), getUserID(4), "Every developer needs to read this!"},
		{getPostID(8), getUserID(6), "The authentication section is particularly useful."},
		{getPostID(8), getUserID(9), "Security should always be a top priority."},

		{getPostID(9), getUserID(1), "As a Spanish speaker, I'd love to explore these places!"},
		{getPostID(9), getUserID(5), "The photography is stunning!"},
		{getPostID(9), getUserID(7), "South America has so much to offer!"},

		{getPostID(10), getUserID(2), "We need more articles like this. The planet needs us!"},
		{getPostID(10), getUserID(6), "Innovation is our best hope for the future."},

		{getPostID(11), getUserID(4), "Following this plan for my first marathon next year!"},
		{getPostID(11), getUserID(8), "The nutrition advice is gold. Thank you!"},
	}

	for _, c := range comments {
		_, err := db.AddComment(c.PostID, c.UserID, c.Text)
		if err != nil {
			log.Printf("Failed to insert comment: %s", c.Text)
		}
	}

	log.Println("All data inserted successfully!")

	log.Println("\n========== ENUM DEMONSTRATION ==========")

	if len(userIDs) > 0 {
		user, err := db.GetUser(userIDs[0])
		if err == nil {
			log.Printf("User Role Type: %T", user.Role)
			log.Printf("User Role Value: %s", user.Role)
			log.Printf("Is Admin Role? %v", user.Role == flash_gen.UserRoleAdmin)
			log.Printf("Is User Role? %v", user.Role == flash_gen.UserRoleUser)
		}
	}

	if len(postIDs) > 0 {
		log.Println("\nAvailable PostStatus enum values:")
		log.Printf("  - %s", flash_gen.PostStatusDraft)
		log.Printf("  - %s", flash_gen.PostStatusPublished)
		log.Printf("  - %s", flash_gen.PostStatusArchived)

		log.Println("\nAvailable UserRole enum values:")
		log.Printf("  - %s", flash_gen.UserRoleAdmin)
		log.Printf("  - %s", flash_gen.UserRoleModerator)
		log.Printf("  - %s", flash_gen.UserRoleUser)
		log.Printf("  - %s", flash_gen.UserRoleGuest)
	}
}
