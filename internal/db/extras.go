package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

func (q *Queries) UpdateGenerationOutput(ctx context.Context, id pgtype.UUID, output, model string) error {
	_, err := q.db.Exec(ctx, `UPDATE generations SET output = $2, model = $3 WHERE id = $1`, id, output, model)
	return err
}

func (q *Queries) UpdateGenerationValidation(ctx context.Context, id pgtype.UUID, validation []byte) error {
	_, err := q.db.Exec(ctx, `UPDATE generations SET validation_result = $2 WHERE id = $1`, id, validation)
	return err
}

func (q *Queries) UpdateStoryTitle(ctx context.Context, id pgtype.UUID, title string) error {
	_, err := q.db.Exec(ctx, `UPDATE stories SET title = $2 WHERE id = $1`, id, title)
	return err
}
