package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run test-helper.go <command> [args...]")
		fmt.Println("Commands:")
		fmt.Println("  verify-connection <db_url>")
		fmt.Println("  count-tables <db_url>")
		fmt.Println("  check-data <db_url> <table> <expected_count>")
		fmt.Println("  verify-migration-table <db_url>")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "verify-connection":
		if len(os.Args) < 3 {
			log.Fatal("Database URL required")
		}
		verifyConnection(os.Args[2])
	case "count-tables":
		if len(os.Args) < 3 {
			log.Fatal("Database URL required")
		}
		countTables(os.Args[2])
	case "check-data":
		if len(os.Args) < 5 {
			log.Fatal("Usage: check-data <db_url> <table> <expected_count>")
		}
		checkData(os.Args[2], os.Args[3], os.Args[4])
	case "verify-migration-table":
		if len(os.Args) < 3 {
			log.Fatal("Database URL required")
		}
		verifyMigrationTable(os.Args[2])
	default:
		log.Fatalf("Unknown command: %s", command)
	}
}

func verifyConnection(dbURL string) {
	fmt.Println("🔗 Verifying database connection...")
	
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	var version string
	err = db.QueryRowContext(ctx, "SELECT version()").Scan(&version)
	if err != nil {
		log.Fatalf("Failed to query version: %v", err)
	}

	fmt.Printf("✅ Database connection successful\n")
	fmt.Printf("📊 PostgreSQL Version: %s\n", version[:50]+"...")
}

func countTables(dbURL string) {
	fmt.Println("📋 Counting database tables...")
	
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT COUNT(*) 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_type = 'BASE TABLE'
	`

	var count int
	err = db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		log.Fatalf("Failed to count tables: %v", err)
	}

	fmt.Printf("✅ Found %d tables in public schema\n", count)

	// List table names
	if count > 0 {
		listQuery := `
			SELECT table_name 
			FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_type = 'BASE TABLE'
			ORDER BY table_name
		`

		rows, err := db.QueryContext(ctx, listQuery)
		if err != nil {
			log.Fatalf("Failed to list tables: %v", err)
		}
		defer rows.Close()

		fmt.Println("📝 Tables:")
		for rows.Next() {
			var tableName string
			if err := rows.Scan(&tableName); err != nil {
				log.Fatalf("Failed to scan table name: %v", err)
			}
			fmt.Printf("   - %s\n", tableName)
		}
	}
}

func checkData(dbURL, tableName, expectedCountStr string) {
	fmt.Printf("🔍 Checking data in table: %s\n", tableName)
	
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	
	var actualCount int
	err = db.QueryRowContext(ctx, query).Scan(&actualCount)
	if err != nil {
		log.Fatalf("Failed to count rows in %s: %v", tableName, err)
	}

	fmt.Printf("✅ Table %s contains %d rows\n", tableName, actualCount)

	// If expected count is provided, verify it
	if expectedCountStr != "any" {
		var expectedCount int
		if _, err := fmt.Sscanf(expectedCountStr, "%d", &expectedCount); err == nil {
			if actualCount != expectedCount {
				log.Fatalf("❌ Expected %d rows, but found %d", expectedCount, actualCount)
			}
			fmt.Printf("✅ Row count matches expected: %d\n", expectedCount)
		}
	}
}

func verifyMigrationTable(dbURL string) {
	fmt.Println("🔍 Verifying migration table...")
	
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if migration table exists
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = '_graft_migrations'
		)
	`

	var exists bool
	err = db.QueryRowContext(ctx, query).Scan(&exists)
	if err != nil {
		log.Fatalf("Failed to check migration table: %v", err)
	}

	if !exists {
		fmt.Println("⚠️  Migration table does not exist (this is normal for fresh installations)")
		return
	}

	// Count migrations
	countQuery := "SELECT COUNT(*) FROM _graft_migrations"
	var migrationCount int
	err = db.QueryRowContext(ctx, countQuery).Scan(&migrationCount)
	if err != nil {
		log.Fatalf("Failed to count migrations: %v", err)
	}

	fmt.Printf("✅ Migration table exists with %d migrations\n", migrationCount)

	// List applied migrations
	if migrationCount > 0 {
		listQuery := `
			SELECT id, migration_name, started_at 
			FROM _graft_migrations 
			ORDER BY started_at
		`

		rows, err := db.QueryContext(ctx, listQuery)
		if err != nil {
			log.Fatalf("Failed to list migrations: %v", err)
		}
		defer rows.Close()

		fmt.Println("📝 Applied migrations:")
		for rows.Next() {
			var id, name string
			var startedAt time.Time
			if err := rows.Scan(&id, &name, &startedAt); err != nil {
				log.Fatalf("Failed to scan migration: %v", err)
			}
			fmt.Printf("   - %s: %s (%s)\n", id, name, startedAt.Format("2006-01-02 15:04:05"))
		}
	}
}
