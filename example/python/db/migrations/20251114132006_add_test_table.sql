-- Migration: add test table
-- Created: 2025-11-14T13:20:06Z

CREATE TABLE test (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO test (name, description, is_active) VALUES 
    ('Test Item 1', 'This is a test item in test_branch', true),
    ('Test Item 2', 'Another test item', true),
    ('Test Item 3', 'Third test item', false);
