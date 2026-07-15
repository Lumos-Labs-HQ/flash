-- ============================================================
-- POSTS queries  (PostgreSQL + UUID primary key)
-- gen_random_uuid() is used by default in the schema.
-- All post id params are UUID type.
-- ============================================================


-- ─── INSERT ──────────────────────────────────────────────────

-- name: CreatePost :one
INSERT INTO posts (user_id, title, body)
VALUES ($1, $2, $3)
RETURNING id, user_id, title, body, views, published, created_at, updated_at;


-- ─── SELECT ──────────────────────────────────────────────────

-- name: GetPost :one
SELECT id, user_id, title, body, views, published, created_at, updated_at
FROM posts
WHERE id = $1
LIMIT 1;


-- name: ListPosts :many
SELECT id, user_id, title, body, views, published, created_at, updated_at
FROM posts
ORDER BY created_at DESC;


-- name: ListPublishedPosts :many
SELECT id, user_id, title, body, views, published, created_at, updated_at
FROM posts
WHERE published = TRUE
ORDER BY created_at DESC;


-- name: ListPostsByUser :many
SELECT id, user_id, title, body, views, published, created_at, updated_at
FROM posts
WHERE user_id = $1
ORDER BY created_at DESC;


-- name: ListPublishedPostsByUser :many
SELECT id, user_id, title, body, views, published, created_at, updated_at
FROM posts
WHERE user_id = $1
  AND published = TRUE
ORDER BY created_at DESC;


-- name: SearchPostsByTitle :many
SELECT id, user_id, title, body, views, published, created_at, updated_at
FROM posts
WHERE title ILIKE '%' || $1 || '%'
ORDER BY created_at DESC;


-- name: ListPostsPaginated :many
SELECT id, user_id, title, body, views, published, created_at, updated_at
FROM posts
ORDER BY created_at DESC
LIMIT $1
OFFSET $2;


-- name: ListPostsCreatedAfter :many
SELECT id, user_id, title, body, views, published, created_at, updated_at
FROM posts
WHERE created_at > $1
ORDER BY created_at ASC;


-- ─── AGGREGATES ──────────────────────────────────────────────

-- name: CountPosts :one
SELECT COUNT(*) AS count FROM posts;


-- name: CountPublishedPosts :one
SELECT COUNT(*) AS count FROM posts WHERE published = TRUE;


-- name: CountPostsByUser :one
SELECT COUNT(*) AS count FROM posts WHERE user_id = $1;


-- name: GetPostViewStats :one
SELECT
    COUNT(*)   AS total_posts,
    SUM(views) AS total_views,
    AVG(views) AS avg_views,
    MAX(views) AS max_views
FROM posts
WHERE published = TRUE;


-- ─── UPDATE ──────────────────────────────────────────────────

-- name: UpdatePostTitle :one
UPDATE posts
SET title      = $1,
    updated_at = NOW()
WHERE id = $2
RETURNING id, user_id, title, body, views, published, created_at, updated_at;


-- name: UpdatePostBody :one
UPDATE posts
SET body       = $1,
    updated_at = NOW()
WHERE id = $2
RETURNING id, user_id, title, body, views, published, created_at, updated_at;


-- name: UpdatePostFull :one
UPDATE posts
SET title      = $1,
    body       = $2,
    published  = $3,
    updated_at = NOW()
WHERE id = $4
RETURNING id, user_id, title, body, views, published, created_at, updated_at;


-- name: PublishPost :one
UPDATE posts
SET published  = TRUE,
    updated_at = NOW()
WHERE id = $1
RETURNING id, user_id, title, body, views, published, created_at, updated_at;


-- name: UnpublishPost :one
UPDATE posts
SET published  = FALSE,
    updated_at = NOW()
WHERE id = $1
RETURNING id, user_id, title, body, views, published, created_at, updated_at;


-- name: IncrementPostViews :one
UPDATE posts
SET views      = views + 1,
    updated_at = NOW()
WHERE id = $1
RETURNING id, user_id, title, body, views, published, created_at, updated_at;


-- ─── DELETE ──────────────────────────────────────────────────

-- name: DeletePost :exec
DELETE FROM posts WHERE id = $1;


-- name: DeletePostsByUser :exec
DELETE FROM posts WHERE user_id = $1;


-- name: DeleteUnpublishedPosts :exec
DELETE FROM posts WHERE published = FALSE;


-- ─── EXISTS ──────────────────────────────────────────────────

-- name: PostExists :one
SELECT EXISTS (
    SELECT 1 FROM posts WHERE id = $1
) AS exists;


-- name: UserHasPost :one
SELECT EXISTS (
    SELECT 1 FROM posts WHERE id = $1 AND user_id = $2
) AS exists;


-- ─── JOIN ────────────────────────────────────────────────────

-- name: GetPostWithAuthor :one
SELECT
    p.id,
    p.title,
    p.body,
    p.views,
    p.published,
    p.created_at,
    u.id        AS author_id,
    u.name      AS author_name,
    u.email     AS author_email
FROM posts p
INNER JOIN users u ON u.id = p.user_id
WHERE p.id = $1;


-- name: ListPublishedPostsWithAuthor :many
SELECT
    p.id,
    p.title,
    p.body,
    p.views,
    p.published,
    p.created_at,
    u.id        AS author_id,
    u.name      AS author_name,
    u.email     AS author_email
FROM posts p
INNER JOIN users u ON u.id = p.user_id
WHERE p.published = TRUE
ORDER BY p.created_at DESC;


-- name: ListPostsWithTags :many
SELECT
    p.id,
    p.title,
    p.published,
    t.name AS tag_name
FROM posts p
INNER JOIN post_tags pt ON pt.post_id = p.id
INNER JOIN tags      t  ON t.id       = pt.tag_id
WHERE p.id = $1
ORDER BY t.name ASC;


-- ─── GROUP BY / HAVING ───────────────────────────────────────

-- name: GetPostCountPerUser :many
SELECT
    user_id,
    COUNT(*) AS post_count
FROM posts
GROUP BY user_id
ORDER BY post_count DESC;


-- name: GetUsersWithMoreThanNPosts :many
SELECT
    user_id,
    COUNT(*) AS post_count
FROM posts
WHERE published = TRUE
GROUP BY user_id
HAVING COUNT(*) > $1
ORDER BY post_count DESC;


-- ─── CASE expression ─────────────────────────────────────────

-- name: ListPostsWithStatus :many
SELECT
    id,
    title,
    views,
    CASE
        WHEN published = TRUE AND views > 1000 THEN 'popular'
        WHEN published = TRUE                  THEN 'published'
        ELSE                                        'draft'
    END AS status
FROM posts
ORDER BY created_at DESC;


-- ─── CTE ─────────────────────────────────────────────────────

-- name: GetTopPostsByViews :many
WITH ranked AS (
    SELECT
        id,
        user_id,
        title,
        views,
        published,
        created_at,
        ROW_NUMBER() OVER (ORDER BY views DESC) AS rank
    FROM posts
    WHERE published = TRUE
)
SELECT id, user_id, title, views, published, created_at, rank
FROM ranked
WHERE rank <= $1;


-- name: GetPostsWithCommentCount :many
WITH comment_counts AS (
    SELECT post_id, COUNT(*) AS comment_count
    FROM comments
    GROUP BY post_id
)
SELECT
    p.id,
    p.title,
    p.published,
    p.created_at,
    COALESCE(cc.comment_count, 0) AS comment_count
FROM posts p
LEFT JOIN comment_counts cc ON cc.post_id = p.id
ORDER BY comment_count DESC;


-- ─── COMPLEX CTEs ────────────────────────────────────────────

-- name: GetPostEngagementSummary :many
-- Multi-step CTE: view score + comment count + tag count combined.
WITH view_scores AS (
    SELECT
        id,
        views,
        NTILE(4) OVER (ORDER BY views DESC) AS view_quartile
    FROM posts
    WHERE published = TRUE
),
comment_counts AS (
    SELECT post_id, COUNT(*) AS comment_count
    FROM comments
    GROUP BY post_id
),
tag_counts AS (
    SELECT post_id, COUNT(*) AS tag_count
    FROM post_tags
    GROUP BY post_id
)
SELECT
    p.id,
    p.title,
    p.views,
    vs.view_quartile,
    COALESCE(cc.comment_count, 0) AS comment_count,
    COALESCE(tc.tag_count,     0) AS tag_count,
    COALESCE(cc.comment_count, 0) + p.views AS engagement_score
FROM posts p
INNER JOIN view_scores   vs ON vs.id      = p.id
LEFT  JOIN comment_counts cc ON cc.post_id = p.id
LEFT  JOIN tag_counts     tc ON tc.post_id = p.id
ORDER BY engagement_score DESC
LIMIT $1;


-- name: GetAuthorLeaderboard :many
-- Recursive-style multi-CTE: published post count, total views,
-- avg comments per post, final rank per author.
WITH author_posts AS (
    SELECT
        user_id,
        COUNT(*)   AS post_count,
        SUM(views) AS total_views
    FROM posts
    WHERE published = TRUE
    GROUP BY user_id
),
author_comments AS (
    SELECT
        p.user_id,
        COUNT(c.id) AS total_comments
    FROM comments c
    INNER JOIN posts p ON p.id = c.post_id
    GROUP BY p.user_id
),
ranked_authors AS (
    SELECT
        ap.user_id,
        ap.post_count,
        ap.total_views,
        COALESCE(ac.total_comments, 0)                              AS total_comments,
        ROUND(COALESCE(ac.total_comments, 0)::NUMERIC /
              NULLIF(ap.post_count, 0), 2)                          AS avg_comments_per_post,
        RANK() OVER (ORDER BY ap.total_views DESC, ap.post_count DESC) AS rank
    FROM author_posts ap
    LEFT JOIN author_comments ac ON ac.user_id = ap.user_id
)
SELECT
    u.id,
    u.name,
    u.email,
    ra.post_count,
    ra.total_views,
    ra.total_comments,
    ra.avg_comments_per_post,
    ra.rank
FROM ranked_authors ra
INNER JOIN users u ON u.id = ra.user_id
ORDER BY ra.rank ASC
LIMIT $1;


-- name: GetPostTrendByDay :many
-- CTE that buckets posts by day and computes a rolling 7-day
-- total using a window function over the daily CTE.
WITH daily AS (
    SELECT
        DATE_TRUNC('day', created_at) AS day,
        COUNT(*)                      AS posts_on_day,
        SUM(views)                    AS views_on_day
    FROM posts
    WHERE published = TRUE
      AND created_at >= $1
    GROUP BY DATE_TRUNC('day', created_at)
)
SELECT
    day,
    posts_on_day,
    views_on_day,
    SUM(posts_on_day) OVER (
        ORDER BY day
        ROWS BETWEEN 6 PRECEDING AND CURRENT ROW
    ) AS rolling_7d_posts,
    SUM(views_on_day) OVER (
        ORDER BY day
        ROWS BETWEEN 6 PRECEDING AND CURRENT ROW
    ) AS rolling_7d_views
FROM daily
ORDER BY day ASC;


-- ─── COMPLEX SUBQUERIES ──────────────────────────────────────

-- name: GetPostsAboveAvgViews :many
-- Correlated subquery: posts whose views beat the global average.
SELECT id, user_id, title, views, published, created_at
FROM posts
WHERE published = TRUE
  AND views > (
      SELECT AVG(views) FROM posts WHERE published = TRUE
  )
ORDER BY views DESC;


-- name: GetMostCommentedPostPerUser :many
-- Correlated subquery in WHERE: for each user, the post with
-- the most comments.
SELECT
    p.id,
    p.user_id,
    p.title,
    p.views,
    p.created_at
FROM posts p
WHERE (
    SELECT COUNT(*)
    FROM comments c
    WHERE c.post_id = p.id
) = (
    SELECT MAX(sub.cnt)
    FROM (
        SELECT COUNT(*) AS cnt
        FROM comments c2
        INNER JOIN posts p2 ON p2.id = c2.post_id
        WHERE p2.user_id = p.user_id
        GROUP BY c2.post_id
    ) sub
)
ORDER BY p.user_id, p.created_at DESC;


-- name: GetPostsWithAllTags :many
-- Relational division: posts that have EVERY tag in a given set.
-- Pass the tag names as an array and the count of that array.
-- e.g. $1 = ARRAY['rust','async']  $2 = 2
SELECT
    p.id,
    p.title,
    p.published,
    p.created_at
FROM posts p
WHERE (
    SELECT COUNT(DISTINCT t.name)
    FROM post_tags pt
    INNER JOIN tags t ON t.id = pt.tag_id
    WHERE pt.post_id = p.id
      AND t.name = ANY($1::TEXT[])
) = $2
ORDER BY p.created_at DESC;


-- name: GetUsersWhoCommentedOnOwnPosts :many
-- Correlated EXISTS: users who left at least one comment on
-- one of their own posts.
SELECT DISTINCT
    u.id,
    u.name,
    u.email
FROM users u
WHERE EXISTS (
    SELECT 1
    FROM comments c
    INNER JOIN posts p ON p.id = c.post_id
    WHERE c.user_id = u.id
      AND p.user_id = u.id
)
ORDER BY u.name ASC;


-- name: GetPostsNotCommentedByUser :many
-- NOT EXISTS subquery: published posts that a specific user
-- has never commented on.
SELECT
    p.id,
    p.title,
    p.views,
    p.created_at
FROM posts p
WHERE p.published = TRUE
  AND NOT EXISTS (
      SELECT 1
      FROM comments c
      WHERE c.post_id = p.id
        AND c.user_id = $1
  )
ORDER BY p.created_at DESC;


-- ─── WINDOW FUNCTIONS ────────────────────────────────────────

-- name: GetPostsWithViewRank :many
-- RANK and LAG over all published posts ordered by views.
SELECT
    id,
    user_id,
    title,
    views,
    created_at,
    RANK()        OVER (ORDER BY views DESC)                         AS view_rank,
    LAG(views, 1) OVER (ORDER BY views DESC)                         AS prev_views,
    views - LAG(views, 1, views) OVER (ORDER BY views DESC)          AS views_diff,
    ROUND(100.0 * views / NULLIF(SUM(views) OVER (), 0), 2)          AS views_pct
FROM posts
WHERE published = TRUE
ORDER BY view_rank ASC;


-- name: GetPerUserPostRanks :many
-- PARTITION BY: rank each user's posts by views within their own posts.
SELECT
    p.id,
    p.user_id,
    u.name      AS author_name,
    p.title,
    p.views,
    p.created_at,
    RANK() OVER (
        PARTITION BY p.user_id
        ORDER BY p.views DESC
    ) AS rank_within_author
FROM posts p
INNER JOIN users u ON u.id = p.user_id
WHERE p.published = TRUE
ORDER BY p.user_id, rank_within_author;


-- name: GetRunningViewTotalByUser :many
-- Running total of views per user ordered by post creation date.
SELECT
    p.id,
    p.user_id,
    p.title,
    p.views,
    p.created_at,
    SUM(p.views) OVER (
        PARTITION BY p.user_id
        ORDER BY p.created_at ASC
        ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
    ) AS running_views
FROM posts p
WHERE p.user_id = $1
ORDER BY p.created_at ASC;
