-- ============================================================
-- Users
-- ============================================================
CREATE TABLE users (
    id         SERIAL       PRIMARY KEY,
    name       TEXT         NOT NULL,
    email      TEXT         UNIQUE NOT NULL,
    age        INTEGER,
    is_active  BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Posts  (one user → many posts)
-- ============================================================
CREATE TABLE posts (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    INTEGER      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title      TEXT         NOT NULL,
    body       TEXT,
    views      INTEGER      NOT NULL DEFAULT 0,
    published  BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Tags  (many posts ↔ many tags via post_tags)
-- ============================================================
CREATE TABLE tags (
    id   SERIAL PRIMARY KEY,
    name TEXT   UNIQUE NOT NULL
);

CREATE TABLE post_tags (
    post_id  UUID    NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    tag_id   INTEGER NOT NULL REFERENCES tags(id)  ON DELETE CASCADE,
    PRIMARY KEY (post_id, tag_id)
);

-- ============================================================
-- Comments  (self-referencing for nested replies)
-- ============================================================
CREATE TABLE comments (
    id         SERIAL       PRIMARY KEY,
    post_id    UUID         NOT NULL REFERENCES posts(id)    ON DELETE CASCADE,
    user_id    INTEGER      NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    parent_id  INTEGER               REFERENCES comments(id) ON DELETE CASCADE,
    body       TEXT         NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Indexes
-- ============================================================
CREATE INDEX idx_posts_user_id    ON posts(user_id);
CREATE INDEX idx_posts_published  ON posts(published);
CREATE INDEX idx_comments_post_id ON comments(post_id);
CREATE INDEX idx_comments_user_id ON comments(user_id);

-- ============================================================
-- PostgreSQL Extensions — pgvector, PostGIS, hstore, ltree
-- ============================================================

-- Products with vector embeddings (pgvector)
CREATE TABLE products (
    id          SERIAL       PRIMARY KEY,
    name        TEXT         NOT NULL,
    description TEXT,
    embedding   VECTOR(1536) NOT NULL,
    half_embed  HALFVEC(768),
    price       MONEY        NOT NULL,
    metadata    HSTORE,
    tags_path   LTREE,
    search_vec  TSVECTOR,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Locations with PostGIS geometry
CREATE TABLE locations (
    id          SERIAL          PRIMARY KEY,
    name        TEXT            NOT NULL,
    coords      GEOMETRY(Point, 4326) NOT NULL,
    area        GEOGRAPHY(Polygon, 4326),
    ip_address  INET,
    mac_addr    MACADDR,
    valid_range TSTZRANGE,
    bit_flags   BIT(8),
    scheduled   INTERVAL,
    raw_xml     XML,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_products_embedding ON products USING ivfflat (embedding vector_cosine_ops);
CREATE INDEX idx_products_tags ON products USING gist (tags_path);
CREATE INDEX idx_products_search ON products USING gin (search_vec);
CREATE INDEX idx_locations_coords ON locations USING gist (coords);
