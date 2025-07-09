-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens(token, created_at, updated_at, user_id, expires_at )
VALUES ($1, NOW(), NOW(), $2, $3) RETURNING *;

-- name: GetRefreshTokenById :one
SELECT * FROM refresh_tokens WHERE token= $1 LIMIT 1;

-- name: RevokeToken :exec
UPDATE refresh_tokens SET revoked_at= NOW(), updated_at= NOW() WHERE token= $1;
