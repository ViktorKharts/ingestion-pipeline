-- name: GetDocument :one
SELECT * FROM documents
WHERE id = ? LIMIT 1;

-- name: ListDocuments :many
SELECT * FROM documents
ORDER BY filename;

-- name: CreateDocument :one
INSERT INTO documents (
  drive_file_id, filename, filepath, content, extension, last_modified, size_bytes
) VALUES (
  ?, ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: SearchDocuments :many
SELECT *
FROM documents
WHERE id IN (
    SELECT rowid FROM documents_fts WHERE documents_fts.content MATCH sqlc.arg(query)
)
LIMIT sqlc.arg(limit);

