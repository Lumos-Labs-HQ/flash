-- Migration: ini
-- Created: 2026-06-14T18:55:45Z

-- +migrate Up
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'post_status') THEN
        CREATE TYPE "post_status" AS ENUM ('draft', 'published', 'archived');
    END IF;
END $$;
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role') THEN
        CREATE TYPE "user_role" AS ENUM ('admin', 'moderator', 'user', 'guest');
    END IF;
END $$;
CREATE TABLE IF NOT EXISTS "categories" (
  "id" SERIAL PRIMARY KEY,
  "name" VARCHAR(255) UNIQUE NOT NULL,
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS "users" (
  "id" SERIAL PRIMARY KEY,
  "name" VARCHAR(255) NOT NULL,
  "address" VARCHAR(255),
  "isadmin" BOOLEAN NOT NULL DEFAULT FALSE,
  "age" INT CHECK (age >= 0),
  "bio" VARCHAR(500),
  "email" VARCHAR(255) UNIQUE NOT NULL,
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "role" user_role NOT NULL DEFAULT 'user'
);
CREATE TABLE IF NOT EXISTS "posts" (
  "id" SERIAL PRIMARY KEY,
  "user_id" INT NOT NULL REFERENCES "users"("id") ON DELETE CASCADE,
  "category_id" INT NOT NULL REFERENCES "categories"("id") ON DELETE SET NULL,
  "title" TEXT NOT NULL,
  "content" TEXT NOT NULL,
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "status" post_status NOT NULL DEFAULT 'draft',
  FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE CASCADE,
  FOREIGN KEY ("category_id") REFERENCES "categories"("id") ON DELETE SET NULL
);
CREATE TABLE IF NOT EXISTS "comments" (
  "id" SERIAL PRIMARY KEY,
  "post_id" INT NOT NULL REFERENCES "posts"("id") ON DELETE CASCADE,
  "user_id" INT NOT NULL REFERENCES "users"("id") ON DELETE CASCADE,
  "content" TEXT NOT NULL,
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  FOREIGN KEY ("post_id") REFERENCES "posts"("id") ON DELETE CASCADE,
  FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE CASCADE
);

-- +migrate Down
DROP TABLE IF EXISTS "comments" CASCADE;
DROP TABLE IF EXISTS "posts" CASCADE;
DROP TABLE IF EXISTS "users" CASCADE;
DROP TABLE IF EXISTS "categories" CASCADE;
DROP TYPE IF EXISTS "user_role";
DROP TYPE IF EXISTS "post_status";
