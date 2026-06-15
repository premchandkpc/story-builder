package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type CreateActorTraitParams struct {
	ActorID    pgtype.UUID `json:"actor_id"`
	TraitKey   string      `json:"trait_key"`
	TraitValue string      `json:"trait_value"`
}

func (q *Queries) CreateActorTrait(ctx context.Context, arg CreateActorTraitParams) (ActorTrait, error) {
	row := q.db.QueryRow(ctx, `
		INSERT INTO actor_traits (actor_id, trait_key, trait_value)
		VALUES ($1, $2, $3)
		RETURNING id, actor_id, trait_key, trait_value, created_at
	`, arg.ActorID, arg.TraitKey, arg.TraitValue)

	var i ActorTrait
	err := row.Scan(&i.ID, &i.ActorID, &i.TraitKey, &i.TraitValue, &i.CreatedAt)
	return i, err
}

func (q *Queries) DeleteActorTraits(ctx context.Context, actorID pgtype.UUID) error {
	_, err := q.db.Exec(ctx, `DELETE FROM actor_traits WHERE actor_id = $1`, actorID)
	return err
}

func (q *Queries) ListActorTraits(ctx context.Context, actorID pgtype.UUID) ([]ActorTrait, error) {
	rows, err := q.db.Query(ctx, `
		SELECT id, actor_id, trait_key, trait_value, created_at
		FROM actor_traits
		WHERE actor_id = $1
		ORDER BY trait_key
	`, actorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ActorTrait
	for rows.Next() {
		var item ActorTrait
		if err := rows.Scan(&item.ID, &item.ActorID, &item.TraitKey, &item.TraitValue, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
