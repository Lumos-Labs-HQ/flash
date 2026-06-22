-- === ENUMS ===

CREATE TYPE post_status AS ENUM ('draft', 'published', 'archived');
CREATE TYPE user_role AS ENUM ('admin', 'moderator', 'user', 'guest');
CREATE TYPE subscription_tier AS ENUM ('free', 'pro', 'enterprise');
CREATE TYPE order_state AS ENUM ('pending', 'confirmed', 'shipped', 'delivered', 'cancelled', 'refunded');

-- === DOMAIN ===

CREATE DOMAIN percentage AS NUMERIC(5,2)
    CHECK (VALUE >= 0 AND VALUE <= 100);

CREATE DOMAIN hex_color AS TEXT
    CHECK (VALUE ~ '^#[0-9a-fA-F]{6}$');

-- === COMPOSITE TYPE ===

CREATE TYPE address_type AS (
    street  TEXT,
    city    TEXT,
    state   TEXT,
    zip     VARCHAR(10),
    country TEXT
);

-- === CORE TABLES ===

CREATE TABLE IF NOT EXISTS users (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    address     VARCHAR(255),
    isadmin     BOOLEAN NOT NULL DEFAULT FALSE,
    age         INT CHECK (age >= 0),
    age_range   INT4RANGE GENERATED ALWAYS AS (
                    CASE WHEN age IS NULL THEN NULL
                         WHEN age < 18  THEN '[0,18)'::int4range
                         WHEN age < 35  THEN '[18,35)'::int4range
                         WHEN age < 55  THEN '[35,55)'::int4range
                         ELSE                '[55,)'::int4range
                    END
                ) STORED,
    bio         VARCHAR(500),
    email       VARCHAR(255) UNIQUE NOT NULL,
    preferences JSONB DEFAULT '{"theme":"light","notifications":true}',
    tags        TEXT[] DEFAULT '{}',
    avatar_hash UUID DEFAULT gen_random_uuid(),
    shipping    address_type,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    role        user_role NOT NULL DEFAULT 'user'
);

CREATE TABLE IF NOT EXISTS categories (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(255) UNIQUE NOT NULL,
    slug        VARCHAR(255) UNIQUE GENERATED ALWAYS AS (lower(regexp_replace(name, '\s+', '-', 'g'))) STORED,
    color       hex_color DEFAULT '#000000',
    metadata    JSONB DEFAULT '{}',
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS posts (
    id              SERIAL PRIMARY KEY,
    user_id         INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id     INT NOT NULL REFERENCES categories(id) ON DELETE SET NULL,
    title           TEXT NOT NULL,
    content         TEXT NOT NULL,
    excerpt         TEXT GENERATED ALWAYS AS (left(content, 200)) STORED,
    tags            TEXT[] DEFAULT '{}',
    metadata        JSONB DEFAULT '{}',
    view_count      BIGINT NOT NULL DEFAULT 0,
    is_featured     BOOLEAN NOT NULL DEFAULT FALSE,
    published_at    TIMESTAMP WITH TIME ZONE,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    status          post_status NOT NULL DEFAULT 'draft',
    CONSTRAINT chk_published_requires_date CHECK (
        status <> 'published' OR published_at IS NOT NULL
    )
);

CREATE TABLE IF NOT EXISTS comments (
    id          SERIAL PRIMARY KEY,
    post_id     INT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id     INT NOT NULL REFERENCES users(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
    parent_id   INT REFERENCES comments(id) ON DELETE CASCADE,
    content     TEXT NOT NULL,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- === ADVANCED TABLES ===

CREATE TABLE IF NOT EXISTS subscriptions (
    id          SERIAL PRIMARY KEY,
    user_id     INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tier        subscription_tier NOT NULL DEFAULT 'free',
    started_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMP WITH TIME ZONE,
    auto_renew  BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE (user_id, tier)            -- one subscription per tier per user
);

CREATE TABLE IF NOT EXISTS orders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         INT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    total_amount    NUMERIC(12,2) NOT NULL CHECK (total_amount > 0),
    discount_pct    percentage DEFAULT 0,
    shipping_addr   address_type NOT NULL,
    line_items      JSONB NOT NULL DEFAULT '[]',
    state           order_state NOT NULL DEFAULT 'pending',
    placed_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS audit_log (
    id          BIGSERIAL PRIMARY KEY,
    table_name  TEXT NOT NULL,
    record_id   TEXT NOT NULL,
    action      TEXT NOT NULL CHECK (action IN ('INSERT', 'UPDATE', 'DELETE')),
    old_data    JSONB,
    new_data    JSONB,
    changed_by  INT REFERENCES users(id),
    changed_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (changed_at);

-- Partition tables
CREATE TABLE audit_log_2024 PARTITION OF audit_log
    FOR VALUES FROM ('2024-01-01') TO ('2025-01-01');
CREATE TABLE audit_log_2025 PARTITION OF audit_log
    FOR VALUES FROM ('2025-01-01') TO ('2026-01-01');
CREATE TABLE audit_log_2026 PARTITION OF audit_log
    FOR VALUES FROM ('2026-01-01') TO ('2027-01-01');

CREATE TABLE IF NOT EXISTS user_sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       TEXT NOT NULL UNIQUE,
    ip_address  INET,
    user_agent  TEXT,
    expires_at  TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- === INDEXES ===

-- Standard B-tree (default)
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_posts_user_id ON posts(user_id);
CREATE INDEX idx_posts_category_id ON posts(category_id);
CREATE INDEX idx_posts_status_created ON posts(status, created_at DESC);
CREATE INDEX idx_comments_post_id_created ON comments(post_id, created_at DESC);
CREATE INDEX idx_comments_user_id ON comments(user_id);

-- Partial index
CREATE INDEX idx_posts_published ON posts(published_at DESC, view_count DESC)
    WHERE status = 'published';

CREATE INDEX idx_comments_top_level ON comments(post_id, created_at)
    WHERE parent_id IS NULL;

-- Expression / functional index
CREATE INDEX idx_users_lower_email ON users(lower(email));
CREATE INDEX idx_posts_title_trgm ON posts USING GIN (title gin_trgm_ops);

-- GiST index
CREATE INDEX idx_users_preferences ON users USING GIN (preferences);
CREATE INDEX idx_posts_metadata ON posts USING GIN (metadata);
CREATE INDEX idx_posts_tags ON posts USING GIN (tags);

-- Unique partial
CREATE UNIQUE INDEX idx_users_active_email ON users(email)
    WHERE isadmin = TRUE;

-- Covering index (INCLUDE)
CREATE INDEX idx_posts_covering ON posts(user_id, status)
    INCLUDE (title, created_at)
    WHERE status = 'published';

-- === VIEWS ===

CREATE VIEW active_users AS
    SELECT id, name, email, role, created_at
    FROM users
    WHERE isadmin = FALSE;

CREATE VIEW user_activity_summary AS
    SELECT
        u.id,
        u.name,
        u.email,
        (SELECT COUNT(*) FROM posts p WHERE p.user_id = u.id)        AS post_count,
        (SELECT COUNT(*) FROM comments c WHERE c.user_id = u.id)     AS comment_count,
        (SELECT MAX(p.created_at) FROM posts p WHERE p.user_id = u.id) AS last_post_at,
        CASE WHEN u.isadmin THEN 'admin'
             WHEN (SELECT COUNT(*) FROM posts p WHERE p.user_id = u.id) > 10 THEN 'power_user'
             ELSE 'regular'
        END AS user_type
    FROM users u;

CREATE MATERIALIZED VIEW post_stats AS
    SELECT
        p.id                 AS post_id,
        p.title,
        COUNT(c.id)          AS comment_count,
        COUNT(DISTINCT c.user_id) AS unique_commenters,
        MAX(c.created_at)    AS last_comment_at
    FROM posts p
    LEFT JOIN comments c ON c.post_id = p.id
    GROUP BY p.id, p.title;

CREATE UNIQUE INDEX idx_post_stats_id ON post_stats(post_id);

-- === TRIGGERS & FUNCTIONS ===

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION log_audit_event()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO audit_log (table_name, record_id, action, new_data, changed_by, changed_at)
        VALUES (TG_TABLE_NAME, NEW.id::TEXT, 'INSERT', to_jsonb(NEW), NEW.user_id, NOW());
        RETURN NEW;
    ELSIF TG_OP = 'UPDATE' THEN
        INSERT INTO audit_log (table_name, record_id, action, old_data, new_data, changed_by, changed_at)
        VALUES (TG_TABLE_NAME, NEW.id::TEXT, 'UPDATE', to_jsonb(OLD), to_jsonb(NEW), NEW.user_id, NOW());
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        INSERT INTO audit_log (table_name, record_id, action, old_data, changed_by, changed_at)
        VALUES (TG_TABLE_NAME, OLD.id::TEXT, 'DELETE', to_jsonb(OLD), OLD.user_id, NOW());
        RETURN OLD;
    END IF;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_posts_updated_at
    BEFORE UPDATE ON posts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- === COMMENTS & DOCUMENTATION ===

COMMENT ON TABLE users IS 'Registered users of the application';
COMMENT ON COLUMN users.email IS 'Unique email address used for authentication';
COMMENT ON COLUMN users.preferences IS 'User preferences as JSONB (theme, notifications, etc.)';
COMMENT ON COLUMN users.tags IS 'User-defined tags for categorization';
COMMENT ON COLUMN users.shipping IS 'Default shipping address (composite type)';

COMMENT ON TABLE posts IS 'Blog posts and articles';
COMMENT ON COLUMN posts.status IS 'Publication status: draft, published, or archived';
COMMENT ON COLUMN posts.published_at IS 'Timestamp when the post was first published';
COMMENT ON COLUMN posts.excerpt IS 'Auto-generated excerpt from first 200 chars of content';

COMMENT ON TABLE audit_log IS 'Partitioned audit trail for all data changes';

-- === NOTIFICATIONS ===

CREATE TABLE IF NOT EXISTS notifications (
    id          BIGSERIAL PRIMARY KEY,
    user_id     INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type        TEXT NOT NULL,
    title       TEXT NOT NULL,
    body        TEXT NOT NULL,
    is_read     BOOLEAN NOT NULL DEFAULT FALSE,
    metadata    JSONB DEFAULT '{}',
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_user_unread ON notifications(user_id, is_read) WHERE is_read = FALSE;
CREATE INDEX idx_notifications_user_created ON notifications(user_id, created_at DESC);

-- === TAGS ===

CREATE TABLE IF NOT EXISTS tags (
    id    SERIAL PRIMARY KEY,
    name  TEXT NOT NULL UNIQUE,
    slug  TEXT NOT NULL UNIQUE,
    color TEXT DEFAULT '#6366f1'
);

CREATE TABLE IF NOT EXISTS post_tags (
    post_id INT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    tag_id  INT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, tag_id)
);

CREATE INDEX idx_post_tags_tag_id ON post_tags(tag_id);

-- === MEDIA ===

CREATE TABLE IF NOT EXISTS media (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    post_id     INT REFERENCES posts(id) ON DELETE SET NULL,
    type        TEXT NOT NULL CHECK (type IN ('image', 'video', 'document')),
    url         TEXT NOT NULL,
    size_bytes  BIGINT NOT NULL DEFAULT 0,
    mime_type   TEXT NOT NULL,
    width       INT,
    height      INT,
    metadata    JSONB DEFAULT '{}',
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_media_user_id ON media(user_id);
CREATE INDEX idx_media_post_id ON media(post_id) WHERE post_id IS NOT NULL;
CREATE INDEX idx_media_type ON media(type, created_at DESC);
