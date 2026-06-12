package db

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func ToUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func FromUUID(u pgtype.UUID) uuid.UUID {
	id, err := uuid.FromBytes(u.Bytes[:])
	if err != nil {
		return uuid.Nil
	}
	return id
}

func JSONBytes(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
