-- Migration: init
-- Created: 2026-06-16T21:35:38Z

-- +migrate Up
CREATE TABLE IF NOT EXISTS "users" (
  "id" ulid PRIMARY KEY DEFAULT gen_ulid(),
  "name" text NOT NULL
);

-- +migrate Down
DROP TABLE IF EXISTS "users" CASCADE;
