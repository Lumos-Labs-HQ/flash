#!/bin/bash

# Graft CLI Demo Script
# This script demonstrates the key features of the Graft CLI tool

set -e

echo "🚀 Graft CLI Tool Demo"
echo "====================="

# Build the tool
echo "📦 Building graft..."
go build -o graft .

echo ""
echo "✅ Graft built successfully!"
echo ""

# Show help
echo "📖 Available commands:"
./graft --help

echo ""
echo "🔧 Testing initialization..."

# Test init (should work)
./graft init || echo "Already initialized"

echo ""
echo "📋 Current project structure:"
ls -la

echo ""
echo "📄 Configuration file:"
cat graft.config.json

echo ""
echo "🆕 Creating a test migration..."
./graft migrate "create test table"

echo ""
echo "📁 Migration files:"
ls -la migrations/

echo ""
echo "📝 Latest migration content:"
cat migrations/*.sql | head -20

echo ""
echo "📊 Migration status (without database):"
echo "Note: This will fail because no DATABASE_URL is set"
./graft status || echo "Expected failure - no database connection"

echo ""
echo "🎯 Demo completed!"
echo ""
echo "To use with a real database:"
echo "1. Set DATABASE_URL environment variable"
echo "2. Run: export DATABASE_URL='postgres://user:pass@localhost:5432/db'"
echo "3. Run: ./graft apply"
echo ""
echo "For development with Docker:"
echo "1. Run: make dev-db"
echo "2. Run: make dev-init"
echo "3. Run: make dev-migrate"
