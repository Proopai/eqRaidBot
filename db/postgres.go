package db

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"
)

func NewPgPool(uri string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.Connect(context.Background(), uri)
	if err != nil {
		return nil, err
	}

	return pool, nil
}
