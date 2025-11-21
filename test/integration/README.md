# FlashORM Integration Tests

Comprehensive integration test suite for FlashORM CLI covering all commands across multiple databases.

## Test Coverage

- **Databases**: PostgreSQL, MySQL, SQLite
- **Commands**: init, migrate, apply, status, gen, pull, export (json/csv/sqlite), raw, studio, reset, branch operations
- **Code Generation**: Go, JavaScript/TypeScript, Python
- **Execution**: Parallel tests for faster execution

## Running Tests Locally

### Prerequisites

- Go 1.23.x or higher
- Docker and Docker Compose
- FlashORM CLI built (`go build -o flash .` from project root)

### Quick Start

```bash
# From the test/integration directory
./run_tests.sh
```

The script will:
1. Start PostgreSQL and MySQL containers via docker-compose
2. Wait for databases to be healthy
3. Run all integration tests in parallel
4. Clean up containers and test artifacts

### Manual Testing

```bash
# Start databases
docker-compose up -d

# Run tests
go test -v -timeout 30m -parallel 3 ./...

# Cleanup
docker-compose down -v
```

### Test Specific Database

```bash
# Run only PostgreSQL tests
go test -v -run TestAllDatabasesParallel/postgresql

# Run only code generation tests
go test -v -run TestCodeGenerationAllLanguages
```

## Running in CI/CD (GitHub Actions)

The tests automatically detect CI environment and skip docker-compose setup. GitHub Actions uses service containers instead:

```yaml
services:
  postgres:
    image: postgres:17-alpine
    # ... health checks and configuration
  
  mysql:
    image: mysql:8.0
    # ... health checks and configuration
```

Environment variables configure database connections:
- `POSTGRES_URL`: PostgreSQL connection string
- `MYSQL_URL`: MySQL connection string
- `CI`: Flag to indicate CI environment

## Test Structure

```
integration/
├── integration_test.go     # Main test suite with all commands
├── codegen_test.go         # Code generation tests for all languages
├── docker-compose.yml      # Local database setup
├── run_tests.sh           # Test runner script
└── README.md              # This file
```

## Configuration

### Database URLs

Tests use environment variables with fallback defaults:

- **PostgreSQL**: `POSTGRES_URL` (default: `postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable`)
- **MySQL**: `MYSQL_URL` (default: `testuser:testpass@tcp(localhost:3306)/testdb`)
- **SQLite**: `SQLITE_URL` (default: `sqlite://./test.db`)

### Timeouts

- Overall test timeout: 30 minutes
- Individual test timeouts vary by complexity
- Database health check timeout: 30 seconds

## Troubleshooting

### Databases not starting

```bash
# Check container status
docker-compose ps

# View logs
docker-compose logs postgres
docker-compose logs mysql

# Force cleanup and restart
docker-compose down -v
docker-compose up -d
```

### Port conflicts

If ports 5432 or 3306 are in use:

```bash
# Find what's using the port
lsof -i :5432
lsof -i :3306

# Stop conflicting services or modify docker-compose.yml ports
```

### Test failures

```bash
# Run tests with verbose output
go test -v -timeout 30m ./...

# Run a specific test
go test -v -run TestAllDatabasesParallel/postgresql/01_Init

# Check test artifacts
ls -la test_projects/
```

## Development

### Adding New Tests

1. Add test function in appropriate file
2. Follow naming convention: `test<Command>(t *testing.T, testDir string, db Database)`
3. Add to test sequence in `testDatabase()` function
4. Update test count in run_tests.sh

### Test Isolation

- Each database test runs in isolated directory under `test_projects/`
- Tests run in parallel using `t.Parallel()`
- Cleanup happens automatically via `defer` statements

## Performance

- Parallel execution reduces total test time by ~60%
- Average runtime: 5-10 minutes locally
- CI runtime: 8-12 minutes (includes setup and teardown)

## Notes

- SQLite tests skip branch operations (not supported)
- Studio tests use different ports for each database to avoid conflicts
- Some commands may produce warnings - check test logs for details
- Test artifacts are automatically cleaned up after tests complete
