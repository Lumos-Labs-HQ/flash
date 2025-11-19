# FlashORM Integration Test Suite

Complete parallel test suite covering all FlashORM commands across all supported databases.

## Test Coverage

### Databases Tested
- ✅ PostgreSQL 16
- ✅ MySQL 8.0
- ✅ SQLite

### Commands Tested

| Command | Description | Test Status |
|---------|-------------|-------------|
| `flash init` | Initialize project | ✅ Tested |
| `flash migrate` | Create migration | ✅ Tested |
| `flash apply` | Apply migrations | ✅ Tested |
| `flash status` | Show migration status | ✅ Tested |
| `flash gen` | Generate code | ✅ Tested |
| `flash pull` | Pull schema | ✅ Tested |
| `flash export --json` | Export to JSON | ✅ Tested |
| `flash export --csv` | Export to CSV | ✅ Tested |
| `flash export --sqlite` | Export to SQLite | ✅ Tested |
| `flash raw` | Execute raw SQL | ✅ Tested |
| `flash studio` | Launch Studio | ✅ Tested |
| `flash reset` | Reset database | ✅ Tested |

### Code Generation Tested
- ✅ Go code generation
- ✅ JavaScript/TypeScript generation
- ✅ Python generation

## Running Tests

### Quick Run
```bash
./run_tests.sh
```

### Manual Run
```bash
# Start databases
docker-compose up -d

# Wait for healthy
sleep 10

# Run tests
go test -v -timeout 10m -parallel 3 ./...

# Cleanup
docker-compose down -v
```

### Run Specific Test
```bash
go test -v -run TestAllDatabasesParallel/postgresql
go test -v -run TestCodeGenerationAllLanguages
```

## Test Architecture

### Parallel Execution
- All 3 databases tested in parallel
- Each database runs in isolated test directory
- No interference between tests

### Docker-Based
- PostgreSQL and MySQL run in Docker containers
- Health checks ensure databases are ready
- Automatic cleanup after tests

### Comprehensive Validation
- File creation verification
- Command output validation
- HTTP endpoint testing (Studio)
- Database connectivity checks

## Test Flow

For each database:
1. **Init** - Create project structure
2. **Migrate** - Create migration file
3. **Apply** - Apply migrations to database
4. **Status** - Verify migration status
5. **Gen** - Generate type-safe code
6. **Pull** - Extract schema from database
7. **Export JSON** - Export data to JSON
8. **Export CSV** - Export data to CSV
9. **Export SQLite** - Export to SQLite (non-SQLite DBs)
10. **Raw** - Execute raw SQL query
11. **Studio** - Launch and verify web UI
12. **Reset** - Reset database

## Requirements

- Go 1.23+
- Docker & Docker Compose
- FlashORM binary built at `../../flash`

## CI/CD Integration

Tests are designed for CI/CD pipelines:
- Non-interactive (--force flags)
- Timeout protection (10m)
- Proper cleanup on failure
- Exit codes for pass/fail
