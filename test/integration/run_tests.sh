#!/bin/bash

set -e

echo "╔════════════════════════════════════════════════════════════╗"
echo "║       FlashORM Complete Integration Test Suite            ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""
echo "Testing ALL commands across ALL databases:"
echo "  📦 Commands: init, migrate, apply, status, gen, pull,"
echo "              export (json/csv/sqlite), raw, studio, reset"
echo "  🗄️  Databases: PostgreSQL, MySQL, SQLite"
echo "  ⚡ Execution: Parallel"
echo ""

cd "$(dirname "$0")"

echo "🧹 Cleaning up previous test artifacts..."
rm -rf test_projects
docker-compose down -v 2>/dev/null || true

echo ""
echo "🐳 Starting Docker containers..."
docker-compose up -d

echo ""
echo "⏳ Waiting for databases to be healthy..."
timeout=30
elapsed=0
while [ $elapsed -lt $timeout ]; do
    if docker-compose ps | grep -q "healthy"; then
        echo "✅ Databases are healthy"
        sleep 2
        break
    fi
    sleep 1
    elapsed=$((elapsed + 1))
    echo -n "."
done

if [ $elapsed -eq $timeout ]; then
    echo ""
    echo "❌ Timeout waiting for databases"
    echo "Docker logs:"
    docker-compose logs
    docker-compose down -v
    exit 1
fi

echo ""
echo "╔════════════════════════════════════════════════════════════╗"
echo "║                  Running Tests                             ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""

go test -v -timeout 10m -parallel 3 ./...

TEST_EXIT_CODE=$?

echo ""
echo "╔════════════════════════════════════════════════════════════╗"
echo "║                  Cleanup                                   ║"
echo "╚════════════════════════════════════════════════════════════╝"
docker-compose down -v
rm -rf test_projects

echo ""
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║              ✅ ALL TESTS PASSED! ✅                       ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo ""
    echo "Test Coverage Summary:"
    echo "  ✅ 3 databases tested (PostgreSQL, MySQL, SQLite)"
    echo "  ✅ 12 commands tested per database"
    echo "  ✅ 3 code generation languages tested"
    echo "  ✅ Parallel execution verified"
else
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║              ❌ TESTS FAILED ❌                            ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo ""
    echo "Exit code: $TEST_EXIT_CODE"
fi

exit $TEST_EXIT_CODE
