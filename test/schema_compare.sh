#!/bin/bash

# Schema Comparison Script
# Usage: ./schema_compare.sh schema1.sql schema2.sql

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if both files are provided
if [ $# -ne 2 ]; then
    echo -e "${RED}Usage: $0 <schema1.sql> <schema2.sql>${NC}"
    exit 1
fi

SCHEMA1="$1"
SCHEMA2="$2"

# Check if files exist
if [ ! -f "$SCHEMA1" ]; then
    echo -e "${RED}Error: $SCHEMA1 not found!${NC}"
    exit 1
fi

if [ ! -f "$SCHEMA2" ]; then
    echo -e "${RED}Error: $SCHEMA2 not found!${NC}"
    exit 1
fi

echo -e "${BLUE}=== SCHEMA COMPARISON REPORT ===${NC}"
echo -e "Schema 1: ${YELLOW}$SCHEMA1${NC}"
echo -e "Schema 2: ${YELLOW}$SCHEMA2${NC}"
echo ""

# Create temp directory
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

# Extract table names from both schemas
echo -e "${BLUE}Extracting table names...${NC}"
grep -i "create table" "$SCHEMA1" | sed 's/create table \([a-zA-Z_][a-zA-Z0-9_]*\).*/\1/i' | sort > "$TEMP_DIR/tables1.txt"
grep -i "create table" "$SCHEMA2" | sed 's/create table \([a-zA-Z_][a-zA-Z0-9_]*\).*/\1/i' | sort > "$TEMP_DIR/tables2.txt"

# Get all unique table names
cat "$TEMP_DIR/tables1.txt" "$TEMP_DIR/tables2.txt" | sort -u > "$TEMP_DIR/all_tables.txt"

CHANGES_FOUND=0
TABLES_COMPARED=0

echo -e "${BLUE}=== TABLE-WISE COMPARISON ===${NC}"

# Compare each table
while IFS= read -r table; do
    TABLES_COMPARED=$((TABLES_COMPARED + 1))
    
    # Check if table exists in both schemas
    if ! grep -q "^$table$" "$TEMP_DIR/tables1.txt"; then
        echo -e "${RED}❌ Table '$table' - MISSING in Schema 1${NC}"
        CHANGES_FOUND=1
        continue
    fi
    
    if ! grep -q "^$table$" "$TEMP_DIR/tables2.txt"; then
        echo -e "${RED}❌ Table '$table' - MISSING in Schema 2${NC}"
        CHANGES_FOUND=1
        continue
    fi
    
    # Extract table structure from both schemas
    awk -v table="$table" '
        BEGIN { IGNORECASE=1; found=0 }
        /create table/ && $3 ~ table { found=1 }
        found { print }
        /^);/ && found { found=0 }
    ' "$SCHEMA1" > "$TEMP_DIR/${table}_schema1.tmp"
    
    awk -v table="$table" '
        BEGIN { IGNORECASE=1; found=0 }
        /create table/ && $3 ~ table { found=1 }
        found { print }
        /^);/ && found { found=0 }
    ' "$SCHEMA2" > "$TEMP_DIR/${table}_schema2.tmp"
    
    # Compare table structures
    if diff -q "$TEMP_DIR/${table}_schema1.tmp" "$TEMP_DIR/${table}_schema2.tmp" > /dev/null; then
        echo -e "${GREEN}✅ Table '$table' - IDENTICAL${NC}"
    else
        echo -e "${YELLOW}⚠️  Table '$table' - CHANGES DETECTED${NC}"
        echo -e "${BLUE}   Differences:${NC}"
        diff -u "$TEMP_DIR/${table}_schema1.tmp" "$TEMP_DIR/${table}_schema2.tmp" | sed 's/^/   /'
        echo ""
        CHANGES_FOUND=1
    fi
    
done < "$TEMP_DIR/all_tables.txt"

echo -e "${BLUE}=== SUMMARY ===${NC}"
echo -e "Tables compared: ${YELLOW}$TABLES_COMPARED${NC}"

if [ $CHANGES_FOUND -eq 0 ]; then
    echo -e "Status: ${GREEN}✅ SCHEMAS ARE IDENTICAL${NC}"
    exit 0
else
    echo -e "Status: ${RED}❌ DIFFERENCES FOUND${NC}"
    exit 1
fi
