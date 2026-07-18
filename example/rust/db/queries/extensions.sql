-- ============================================================
-- PRODUCTS queries (pgvector, hstore, ltree, tsvector, money)
-- ============================================================

-- name: GetProduct :one
SELECT id, name, description, embedding, half_embed, price, metadata, tags_path, search_vec, created_at
FROM products
WHERE id = $1;

-- name: ListProducts :many
SELECT id, name, description, embedding, half_embed, price, metadata, tags_path, search_vec, created_at
FROM products
ORDER BY created_at DESC;

-- name: CreateProduct :one
INSERT INTO products (name, description, embedding, half_embed, price, metadata, tags_path)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, name, description, embedding, half_embed, price, metadata, tags_path, search_vec, created_at;

-- name: SearchSimilarProducts :many
-- pgvector: find nearest neighbors by cosine distance
SELECT id, name, description, price, embedding <=> $1 AS distance
FROM products
ORDER BY embedding <=> $1
LIMIT $2;

-- name: SearchProductsByText :many
-- Full-text search using tsvector
SELECT id, name, description, price, ts_rank(search_vec, plainto_tsquery('english', $1)) AS rank
FROM products
WHERE search_vec @@ plainto_tsquery('english', $1)
ORDER BY rank DESC;

-- name: GetProductsByPath :many
-- ltree: find all products under a category path
SELECT id, name, price, tags_path
FROM products
WHERE tags_path <@ $1::LTREE
ORDER BY tags_path;

-- name: GetProductMetadata :one
-- hstore: get metadata for a product
SELECT id, name, metadata
FROM products
WHERE id = $1;

-- name: UpdateProductEmbedding :exec
-- pgvector: update embedding
UPDATE products
SET embedding = $1
WHERE id = $2;

-- name: CountProductsInPriceRange :one
-- money: count products in price range
SELECT COUNT(*) AS count
FROM products
WHERE price BETWEEN $1 AND $2;


-- ============================================================
-- LOCATIONS queries (PostGIS, inet, macaddr, ranges, bit, interval, xml)
-- ============================================================

-- name: GetLocation :one
SELECT id, name, coords, area, ip_address, mac_addr, valid_range, bit_flags, scheduled, raw_xml, created_at
FROM locations
WHERE id = $1;

-- name: ListLocations :many
SELECT id, name, coords, area, ip_address, mac_addr, valid_range, bit_flags, scheduled, raw_xml, created_at
FROM locations
ORDER BY created_at DESC;

-- name: CreateLocation :one
INSERT INTO locations (name, coords, area, ip_address, mac_addr, valid_range, bit_flags, scheduled, raw_xml)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, name, coords, area, ip_address, mac_addr, valid_range, bit_flags, scheduled, raw_xml, created_at;

-- name: FindNearbyLocations :many
-- PostGIS: find locations within radius (meters)
SELECT id, name, ST_Distance(coords::geography, $1::geography) AS distance_meters
FROM locations
WHERE ST_DWithin(coords::geography, $1::geography, $2)
ORDER BY distance_meters ASC;

-- name: FindLocationsInArea :many
-- PostGIS: find locations within a polygon
SELECT id, name, coords, created_at
FROM locations
WHERE ST_Within(coords, $1::geometry);

-- name: GetLocationsByNetwork :many
-- inet: find locations by IP subnet
SELECT id, name, ip_address, created_at
FROM locations
WHERE ip_address <<= $1::INET
ORDER BY ip_address;

-- name: GetLocationsValidAt :many
-- tstzrange: find locations valid at a given timestamp
SELECT id, name, valid_range, created_at
FROM locations
WHERE valid_range @> $1::TIMESTAMPTZ
ORDER BY name;

-- name: UpdateLocationSchedule :exec
-- interval: update schedule
UPDATE locations
SET scheduled = $1
WHERE id = $2;

-- name: DeleteLocation :exec
DELETE FROM locations WHERE id = $1;
