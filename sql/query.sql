-- name: GetRandomMessage :one
SELECT id, content, created_at 
FROM messages 
ORDER BY RANDOM() 
LIMIT 1;

-- name: ListRandomMessages :many
SELECT id, content, created_at
FROM messages
ORDER BY RANDOM()
LIMIT ?;

-- name: ListPendingMessages :many
SELECT id, content, status, created_at 
FROM pending_messages 
WHERE status = 'pending'
ORDER BY created_at DESC;

-- name: CreatePendingMessage :one
INSERT INTO pending_messages (content)
VALUES (?)
RETURNING id;

-- name: GetPendingMessage :one
SELECT id, content, status, created_at
FROM pending_messages
WHERE id = ?;

-- name: CreateMessage :one
INSERT INTO messages (content)
VALUES (?)
RETURNING id;

-- name: UpdatePendingMessageStatus :exec
UPDATE pending_messages 
SET status = ?2
WHERE id = ?1;
