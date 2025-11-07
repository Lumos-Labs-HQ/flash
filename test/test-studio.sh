#!/bin/bash

# Test script for FlashORM Studio

echo "🧪 Testing FlashORM Studio Setup"
echo "=============================="

# Set DATABASE_URL (edit this with your database connection)
export DATABASE_URL="postgresql://neondb_owner:npg_m7N0XzUxPnMu@ep-spring-hat-a8qw0bvp-pooler.eastus2.azure.neon.tech/neondb?sslmode=require&channel_binding=require"

# Uncomment and edit one of these based on your database:
# export DATABASE_URL="postgres://user:password@localhost:5432/dbname"
# export DATABASE_URL="mysql://user:password@localhost:3306/dbname"
# export DATABASE_URL="sqlite://./test.db"

echo "📝 DATABASE_URL set to: $DATABASE_URL"
echo ""

# Check if binary exists
if [ ! -f "./FlashORM.exe" ]; then
    echo "❌ FlashORM.exe not found. Building..."
    go build -o FlashORM.exe .
fi

# Check if templates directory exists
if [ ! -d "./web/studio/templates" ]; then
    echo "❌ Templates directory not found"
    exit 1
fi

# Check if static files exist
if [ ! -d "./web/studio/static" ]; then
    echo "❌ Static files directory not found"
    exit 1
fi

echo "✅ All files present"
echo ""
echo "🚀 Starting FlashORM Studio..."
echo "   Browser will open at http://localhost:5555"
echo ""
echo "Press Ctrl+C to stop"
echo ""

# Run studio
./FlashORM.exe studio
