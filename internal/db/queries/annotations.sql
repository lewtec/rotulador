-- name: CreateAnnotation :one
INSERT INTO annotations (image_sha256, username, stage_index, option_value)
VALUES (?, ?, ?, ?)
ON CONFLICT(image_sha256, username, stage_index)
DO UPDATE SET
  option_value = excluded.option_value,
  annotated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetAnnotation :one
SELECT * FROM annotations
WHERE image_sha256 = ? AND username = ? AND stage_index = ?;

-- name: GetAnnotationsForImage :many
SELECT * FROM annotations
WHERE image_sha256 = ?
ORDER BY stage_index ASC;

-- name: GetAnnotationsByUser :many
SELECT a.*, i.filename
FROM annotations a
JOIN images i ON a.image_sha256 = i.sha256
WHERE a.username = ?
ORDER BY a.annotated_at DESC
LIMIT ? OFFSET ?;

-- name: GetAnnotationsByImageAndUser :many
SELECT * FROM annotations
WHERE image_sha256 = ? AND username = ?
ORDER BY stage_index ASC;

-- name: CountAnnotationsByUser :one
SELECT COUNT(*) FROM annotations
WHERE username = ?;

-- name: ListPendingImagesForUserAndStage :many
WITH annotated_images AS (
  SELECT image_sha256 FROM annotations WHERE username = ? AND stage_index = ?
)
SELECT i.*
FROM images i
LEFT JOIN annotated_images ai ON i.sha256 = ai.image_sha256
WHERE ai.image_sha256 IS NULL
ORDER BY i.filename ASC
LIMIT ?;

-- name: CheckAnnotationExists :one
SELECT CAST(EXISTS (
    SELECT 1
    FROM annotations
    WHERE image_sha256 = ? AND username = ? AND stage_index = ?
) AS INTEGER);

-- name: DeleteAnnotation :exec
DELETE FROM annotations
WHERE id = ?;

-- name: DeleteAnnotationsForImage :exec
DELETE FROM annotations
WHERE image_sha256 = ?;

-- name: GetAnnotationStats :one
SELECT
  COUNT(DISTINCT image_sha256) as annotated_images,
  COUNT(*) as total_annotations,
  COUNT(DISTINCT username) as total_users
FROM annotations;

-- name: CountPendingImagesForUserAndStage :one
WITH annotated_images AS (
  SELECT image_sha256 FROM annotations WHERE username = ? AND stage_index = ?
)
SELECT COUNT(*)
FROM images i
LEFT JOIN annotated_images ai ON i.sha256 = ai.image_sha256
WHERE ai.image_sha256 IS NULL;

-- name: GetImageHashesWithAnnotation :many
SELECT DISTINCT image_sha256
FROM annotations
WHERE stage_index = ? AND option_value = ?;

-- name: CheckAnnotationExistsForImageStage :one
SELECT CAST(EXISTS (
    SELECT 1
    FROM annotations
    WHERE image_sha256 = ? AND stage_index = ?
) AS INTEGER);

-- name: CountImagesWithoutAnnotationForStage :one
WITH annotated_images AS (
  SELECT DISTINCT image_sha256 FROM annotations WHERE stage_index = ?
)
SELECT COUNT(*)
FROM images i
LEFT JOIN annotated_images ai ON i.sha256 = ai.image_sha256
WHERE ai.image_sha256 IS NULL;

-- name: CountImagesWithAnnotation :one
SELECT COUNT(DISTINCT image_sha256)
FROM annotations
WHERE stage_index = ? AND option_value = ?;

-- name: GetAllImageSHA256s :many
SELECT sha256 FROM images ORDER BY sha256;

-- name: GetAnnotationsForStageAndValue :many
SELECT image_sha256, username, annotated_at
FROM annotations
WHERE stage_index = ? AND option_value = ?
ORDER BY image_sha256;

-- name: GetImagesWithoutAnnotationForStage :many
SELECT i.sha256, i.filename
FROM images i
WHERE NOT EXISTS (
    SELECT 1 FROM annotations a
    WHERE a.image_sha256 = i.sha256 AND a.stage_index = ?
)
ORDER BY i.filename;

-- name: CountImagesWithAnnotationInList :one
SELECT COUNT(DISTINCT image_sha256)
FROM annotations
WHERE stage_index = ? AND option_value = ?
  AND image_sha256 IN (sqlc.slice('image_hashes'));
