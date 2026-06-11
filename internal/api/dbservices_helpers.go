package api

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func toUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func fromUUID(u pgtype.UUID) uuid.UUID {
	id, _ := uuid.FromBytes(u.Bytes[:])
	return id
}

func jsonBytes(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
