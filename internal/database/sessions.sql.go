package database

import (
	"context"

	"github.com/google/uuid"
)

const updateSessionStatus = `-- name: UpdateSessionStatus :exec
UPDATE sessions 
SET status=$1
WHERE id=$2
`

type UpdateSessionStatusParams struct {
	Status string
	ID     uuid.UUID
}

func (q *Queries) UpdateSessionStatus(ctx context.Context, arg UpdateSessionStatusParams) error {
	_, err := q.db.ExecContext(ctx, updateSessionStatus, arg.Status, arg.ID)
	return err
}

const getSession = `-- name: GetSession :one
SELECT id, created_at, name, user_id, status, job_title, job_description FROM sessions 
WHERE id = $1
`

func (q *Queries) GetSession(ctx context.Context, id uuid.UUID) (Session, error) {
	row := q.db.QueryRowContext(ctx, getSession, id)
	var i Session
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.Name,
		&i.UserID,
		&i.Status,
		&i.JobTitle,
		&i.JobDescription,
	)
	return i, err
}
