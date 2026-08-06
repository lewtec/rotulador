-- name: CreateImage :one
INSERT INTO images (sha256, filename)
VALUES (?, ?)
ON CONFLICT(sha256) DO UPDATE SET filename = excluded.filename
RETURNING *;

-- name: GetImage :one
SELECT * FROM images
WHERE sha256 = ?;

-- name: GetImageByFilename :one
SELECT * FROM images
WHERE filename = ?;

-- name: ListImages :many
SELECT * FROM images
ORDER BY filename;

-- name: ListImagesNotFinished :many
SELECT * FROM images
ORDER BY filename ASC
LIMIT ?;

-- name: CountImages :one
SELECT COUNT(*) FROM images;

-- name: DeleteImage :exec
DELETE FROM images
WHERE sha256 = ?;
