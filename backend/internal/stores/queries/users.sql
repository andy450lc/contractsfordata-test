-- name: GetUser :one
SELECT id, email, name, created_at, updated_at
FROM users
WHERE id = $1;

-- name: UpsertUserIfNewer :one
INSERT INTO users (id, email, name, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE
  SET email = excluded.email,
      name = excluded.name,
      updated_at = excluded.updated_at
  WHERE excluded.updated_at >= users.updated_at
RETURNING id, email, name, created_at, updated_at;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;
