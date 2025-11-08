package database

import (
	"context"

	"github.com/google/uuid"
)

const getResumesBySession = `-- name: GetResumesBySession :many
SELECT id, original_filename, mime, size_bytes, storage_provider, object_key, storage_url, upload_status, created_at, session_id FROM resumes WHERE session_id=$1
`

func (q *Queries) GetResumesBySession(ctx context.Context, sessionID uuid.UUID) ([]Resume, error) {
	rows, err := q.db.QueryContext(ctx, getResumesBySession, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Resume
	for rows.Next() {
		var i Resume
		if err := rows.Scan(
			&i.ID,
			&i.OriginalFilename,
			&i.Mime,
			&i.SizeBytes,
			&i.StorageProvider,
			&i.ObjectKey,
			&i.StorageUrl,
			&i.UploadStatus,
			&i.CreatedAt,
			&i.SessionID,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
