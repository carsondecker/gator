-- name: CreateFeed :one
INSERT INTO feeds (id, created_at, updated_at, name, url, user_id)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING *;

-- name: GetFeeds :many
SELECT * FROM feeds;

-- name: GetFeedsWithUser :many
SELECT f.id, f.name, f.url, u.name AS username
FROM feeds f
    JOIN users u
        ON f.user_id = u.id;