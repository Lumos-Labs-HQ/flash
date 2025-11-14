-- Migration: add inventory table from schema
-- Created: 2025-11-14T13:29:24Z

CREATE TABLE IF NOT EXISTS inventory (
  id SERIAL PRIMARY KEY,
  product_name VARCHAR(255) NOT NULL,
  quantity INT NOT NULL DEFAULT 0,
  price DECIMAL(10,2) NOT NULL,
  warehouse_location VARCHAR(100),
  last_updated TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

INSERT INTO inventory (product_name, quantity, price, warehouse_location) VALUES
  ('Widget A', 100, 25.99, 'Warehouse 1'),
  ('Widget B', 50, 45.50, 'Warehouse 2'),
  ('Gadget X', 200, 15.75, 'Warehouse 1');
