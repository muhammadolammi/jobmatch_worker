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
