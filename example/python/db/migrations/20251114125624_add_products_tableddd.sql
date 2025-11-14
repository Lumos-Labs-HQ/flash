-- Migration: add products table
-- Created: 2025-11-14T12:56:24Z

CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    price DECIMAL(10,2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO products (name, price) VALUES 
    ('Laptop', 999.99),
    ('Mouse', 29.99),
    ('Keyboard', 79.99);
