-- Migration: add orders table
-- Created: 2025-11-14T13:05:48Z

CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer_name VARCHAR(255) NOT NULL,
    total_amount DECIMAL(10,2),
    status VARCHAR(50) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO orders (customer_name, total_amount, status) VALUES 
    ('John Doe', 1500.00, 'completed'),
    ('Jane Smith', 750.50, 'pending'),
    ('Bob Johnson', 2200.00, 'shipped');
