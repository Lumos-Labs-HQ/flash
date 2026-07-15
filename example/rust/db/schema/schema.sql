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
