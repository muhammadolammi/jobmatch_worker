package database

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

const createOrUpdateAnalysesResults = `-- name: CreateOrUpdateAnalysesResults :exec
INSERT INTO analyses_results (
results, session_id)
VALUES ( $1, $2)
ON CONFLICT (session_id)
DO UPDATE SET
    results = EXCLUDED.results,
    updated_at = CURRENT_TIMESTAMP
`

type CreateOrUpdateAnalysesResultsParams struct {
	Results   json.RawMessage
	SessionID uuid.UUID
}

func (q *Queries) CreateOrUpdateAnalysesResults(ctx context.Context, arg CreateOrUpdateAnalysesResultsParams) error {
	_, err := q.db.ExecContext(ctx, createOrUpdateAnalysesResults, arg.Results, arg.SessionID)
	return err
}
