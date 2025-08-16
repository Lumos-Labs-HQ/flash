#!/bin/bash

# Database helper functions for CI/CD

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

log() {
    echo -e "${GREEN}[DB-HELPER]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Wait for PostgreSQL to be ready
wait_for_postgres() {
    local host=${1:-localhost}
    local port=${2:-5432}
    local user=${3:-postgres}
    local max_attempts=${4:-30}
    
    log "Waiting for PostgreSQL at $host:$port..."
    
    for i in $(seq 1 $max_attempts); do
        if pg_isready -h "$host" -p "$port" -U "$user" >/dev/null 2>&1; then
            log "PostgreSQL is ready!"
            return 0
        fi
        
        echo -n "."
        sleep 1
    done
    
    error "PostgreSQL failed to start after $max_attempts attempts"
}

# Test database connection
test_connection() {
    local db_url="$1"
    
    if [ -z "$db_url" ]; then
        error "Database URL is required"
    fi
    
    log "Testing database connection..."
    
    # Extract connection details from URL
    if echo "$db_url" | grep -qE "(postgres|postgresql)://"; then
        log "PostgreSQL connection detected"
        
        if command -v psql >/dev/null 2>&1; then
            if psql "$db_url" -c "SELECT 1;" >/dev/null 2>&1; then
                log "Database connection successful"
                return 0
            else
                error "Failed to connect to database"
            fi
        else
            log "psql not available, skipping direct connection test"
        fi
    else
        error "Unsupported database URL format"
    fi
}

# Create test data SQL file
create_test_data() {
    local file_path="$1"
    
    cat > "$file_path" << 'EOF'
-- Test data for integration tests
INSERT INTO users (name, email) VALUES 
('Alice Johnson', 'alice@test.com'),
('Bob Wilson', 'bob@test.com'),
('Charlie Brown', 'charlie@test.com'),
('Diana Prince', 'diana@test.com'),
('Eve Adams', 'eve@test.com');
EOF
    
    log "Test data file created: $file_path"
}

# Create complex migration SQL
create_complex_migration() {
    local file_path="$1"
    
    cat > "$file_path" << 'EOF'
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    content TEXT,
    published BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_posts_user_id ON posts(user_id);
CREATE INDEX idx_posts_published ON posts(published) WHERE published = TRUE;

ALTER TABLE users ADD CONSTRAINT check_email_format 
    CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$');
EOF
    
    log "Complex migration file created: $file_path"
}

# Verify table structure
verify_table_structure() {
    local db_url="$1"
    local table_name="$2"
    
    if command -v psql >/dev/null 2>&1; then
        log "Verifying table structure for: $table_name"
        
        # Check if table exists and get column info
        psql "$db_url" -c "\d $table_name" >/dev/null 2>&1 || error "Table $table_name does not exist"
        
        log "Table $table_name structure verified"
    else
        log "psql not available, skipping table structure verification"
    fi
}

# Main function to handle commands
main() {
    case "$1" in
        "wait")
            wait_for_postgres "$2" "$3" "$4" "$5"
            ;;
        "test-connection")
            test_connection "$2"
            ;;
        "create-test-data")
            create_test_data "$2"
            ;;
        "create-complex-migration")
            create_complex_migration "$2"
            ;;
        "verify-table")
            verify_table_structure "$2" "$3"
            ;;
        *)
            echo "Usage: $0 {wait|test-connection|create-test-data|create-complex-migration|verify-table}"
            echo ""
            echo "Commands:"
            echo "  wait [host] [port] [user] [max_attempts]  - Wait for PostgreSQL"
            echo "  test-connection <db_url>                  - Test database connection"
            echo "  create-test-data <file_path>              - Create test data SQL"
            echo "  create-complex-migration <file_path>      - Create complex migration"
            echo "  verify-table <db_url> <table_name>       - Verify table structure"
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"
