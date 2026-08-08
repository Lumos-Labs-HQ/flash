-- Migration: init2
-- Created: 2026-08-08T22:51:50Z

-- +migrate Up
CREATE TABLE IF NOT EXISTS "categories" (
  "id" SERIAL PRIMARY KEY,
  "name" TEXT UNIQUE NOT NULL,
  "description" TEXT,
  "parent_id" INTEGER REFERENCES "categories"("id"),
  "sort_order" INTEGER NOT NULL DEFAULT 0,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS "post_categories" (
  "post_id" UUID NOT NULL REFERENCES "posts"("id") ON DELETE CASCADE,
  "category_id" INTEGER NOT NULL REFERENCES "categories"("id") ON DELETE CASCADE,
  PRIMARY KEY ("post_id", "category_id")
);

-- +migrate Down
DROP TABLE IF EXISTS "post_categories" CASCADE;
DROP TABLE IF EXISTS "categories" CASCADE;