# Quick Start - Integration Tests

## Run All Tests

```bash
# From project root
make test-integration

# Or directly
cd test/integration
./run_tests.sh
```

## What Gets Tested

### ✅ All 12 Commands
1. `flash init` - Project initialization
2. `flash migrate` - Migration creation
3. `flash apply` - Apply migrations
4. `flash status` - Migration status
5. `flash gen` - Code generation
6. `flash pull` - Schema extraction
7. `flash export --json` - JSON export
8. `flash export --csv` - CSV export
9. `flash export --sqlite` - SQLite export
10. `flash raw` - Raw SQL execution
11. `flash studio` - Web UI launch
12. `flash reset` - Database reset

### ✅ All 3 Databases
- PostgreSQL 16
- MySQL 8.0
- SQLite

### ✅ All 3 Code Generators
- Go
- JavaScript/TypeScript
- Python

## Test Execution

- **Parallel**: All 3 databases tested simultaneously
- **Isolated**: Each database in separate directory
- **Docker**: PostgreSQL & MySQL in containers
- **Fast**: ~2-3 minutes total

## Requirements

```bash
# Check requirements
docker --version          # Docker 20.10+
docker-compose --version  # Docker Compose 1.29+
go version               # Go 1.23+
```

## Manual Test Run

```bash
# 1. Start databases
docker-compose up -d

# 2. Wait for healthy
sleep 10

# 3. Run tests
go test -v -timeout 10m -parallel 3 ./...

# 4. Cleanup
docker-compose down -v
rm -rf test_projects
```

## Run Specific Database

```bash
go test -v -run TestAllDatabasesParallel/postgresql
go test -v -run TestAllDatabasesParallel/mysql
go test -v -run TestAllDatabasesParallel/sqlite
```

## Run Specific Command

```bash
go test -v -run TestAllDatabasesParallel/postgresql/01_Init
go test -v -run TestAllDatabasesParallel/mysql/11_Studio
```

## Troubleshooting

### Docker not starting
```bash
docker-compose down -v
docker system prune -f
docker-compose up -d
```

### Port conflicts
```bash
# Check ports
lsof -i :5432  # PostgreSQL
lsof -i :3306  # MySQL

# Kill processes or change ports in docker-compose.yml
```

### Tests hanging
```bash
# Kill stuck processes
pkill -f flash
docker-compose down -v
rm -rf test_projects
```

## CI/CD Integration

```yaml
# GitHub Actions example
- name: Run Integration Tests
  run: make test-integration
```

## Expected Output

```
╔════════════════════════════════════════════════════════════╗
║       FlashORM Complete Integration Test Suite            ║
╚════════════════════════════════════════════════════════════╝

Testing ALL commands across ALL databases:
  📦 Commands: init, migrate, apply, status, gen, pull,
              export (json/csv/sqlite), raw, studio, reset
  🗄️  Databases: PostgreSQL, MySQL, SQLite
  ⚡ Execution: Parallel

🐳 Starting Docker containers...
⏳ Waiting for databases to be healthy...
✅ Databases are healthy

╔════════════════════════════════════════════════════════════╗
║                  Running Tests                             ║
╚════════════════════════════════════════════════════════════╝

=== RUN   TestAllDatabasesParallel
=== PAUSE TestAllDatabasesParallel
=== CONT  TestAllDatabasesParallel
=== RUN   TestAllDatabasesParallel/postgresql
=== PAUSE TestAllDatabasesParallel/postgresql
=== RUN   TestAllDatabasesParallel/mysql
=== PAUSE TestAllDatabasesParallel/mysql
=== RUN   TestAllDatabasesParallel/sqlite
=== PAUSE TestAllDatabasesParallel/sqlite
...
--- PASS: TestAllDatabasesParallel (45.23s)
    --- PASS: TestAllDatabasesParallel/postgresql (42.15s)
    --- PASS: TestAllDatabasesParallel/mysql (43.87s)
    --- PASS: TestAllDatabasesParallel/sqlite (38.92s)
PASS
ok      github.com/Lumos-Labs-HQ/flash/test/integration 45.456s

╔════════════════════════════════════════════════════════════╗
║              ✅ ALL TESTS PASSED! ✅                       ║
╚════════════════════════════════════════════════════════════╝

Test Coverage Summary:
  ✅ 3 databases tested (PostgreSQL, MySQL, SQLite)
  ✅ 12 commands tested per database
  ✅ 3 code generation languages tested
  ✅ Parallel execution verified
```
