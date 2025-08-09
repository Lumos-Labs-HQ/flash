-- Migration: added the password
-- Created: 2025-08-09 18:06:20

ALTER TABLE "users" ADD COLUMN IF NOT EXISTS "password_hash" VARCHAR(255) NOT NULL;