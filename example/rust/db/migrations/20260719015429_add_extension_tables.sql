-- Migration: add extension tables
-- Created: 2026-07-19T01:54:29Z

-- +migrate Up
CREATE TABLE IF NOT EXISTS "locations" (
  "id" SERIAL PRIMARY KEY,
  "name" TEXT NOT NULL,
  "coords" GEOMETRY(Point, 4326) NOT NULL,
  "area" GEOGRAPHY(Polygon, 4326),
  "ip_address" INET,
  "mac_addr" MACADDR,
  "valid_range" TSTZRANGE,
  "bit_flags" BIT(8),
  "scheduled" INTERVAL,
  "raw_xml" XML,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX "idx_locations_coords" ON "locations" USING GIST (coords);

CREATE TABLE IF NOT EXISTS "products" (
  "id" SERIAL PRIMARY KEY,
  "name" TEXT NOT NULL,
  "description" TEXT,
  "embedding" VECTOR(1536) NOT NULL,
  "half_embed" HALFVEC(768),
  "price" MONEY NOT NULL,
  "metadata" HSTORE,
  "tags_path" LTREE,
  "search_vec" TSVECTOR,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX "idx_products_embedding" ON "products" USING IVFFLAT (embedding vector_cosine_ops);

CREATE INDEX "idx_products_tags" ON "products" USING GIST (tags_path);

CREATE INDEX "idx_products_search" ON "products" USING GIN (search_vec);

-- +migrate Down
DROP INDEX IF EXISTS "idx_products_search";
DROP INDEX IF EXISTS "idx_products_tags";
DROP INDEX IF EXISTS "idx_products_embedding";
DROP TABLE IF EXISTS "products" CASCADE;
DROP INDEX IF EXISTS "idx_locations_coords";
DROP TABLE IF EXISTS "locations" CASCADE;