package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

func (q *Queries) UpdateGenerationOutput(ctx context.Context, id pgtype.UUID, output, model string) error {
	_, err := q.db.Exec(ctx, `UPDATE generations SET output = $2, model = $3 WHERE id = $1`, id, output, model)
	return err
}
