#!/bin/bash

set -e

echo "╔════════════════════════════════════════════════════════════╗"
echo "║       FlashORM Complete Integration Test Suite            ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""

cd "$(dirname "$0")"

# Detect if running in CI or local environment
if [ -n "$CI" ] || [ -n "$GITHUB_ACTIONS" ]; then
    echo "🔄 CI Mode: Using GitHub Actions service containers"
    echo ""
else
    echo "📦 Local Mode: Using docker-compose"
    echo ""
    echo "Testing ALL commands across ALL databases:"
    echo "  📦 Commands: init, migrate, apply, status, gen, pull,"
    echo "              export (json/csv/sqlite), raw, studio, reset"
    echo "  🗄️  Databases: PostgreSQL, MySQL, SQLite"
    echo "  ⚡ Execution: Parallel"
    echo ""

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
        if docker-compose ps --format json 2>/dev/null | grep -q "healthy" || \
           docker-compose ps 2>/dev/null | grep -q "healthy"; then
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
fi
echo "╔════════════════════════════════════════════════════════════╗"
echo "║                  Running Tests                             ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""

go test -v -timeout 30m -parallel 3 ./...

TEST_EXIT_CODE=$?

echo ""
echo "╔════════════════════════════════════════════════════════════╗"
echo "║                  Cleanup                                   ║"
echo "╚════════════════════════════════════════════════════════════╝"

# Only cleanup docker-compose if running locally
if [ -z "$CI" ] && [ -z "$GITHUB_ACTIONS" ]; then
    echo "🧹 Stopping docker-compose services..."
    docker-compose down -v
fi

rm -rf test_projects

echo ""
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║              ✅ ALL TESTS PASSED! ✅                       ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo ""
    echo "Test Coverage Summary:"
    echo "  ✅ 3 databases tested (PostgreSQL, MySQL, SQLite)"
    echo "  ✅ 17 commands tested per database"
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
