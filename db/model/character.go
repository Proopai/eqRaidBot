package model

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

type Character struct {
	Id        int64
	Name      string
	Class     int64
	Level     int64
	AA        int64
	Bot       bool
	CreatedBy string
	CreatedAt time.Time
}

func (r *Character) Save(db *pgxpool.Pool) error {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return err
	}

	defer conn.Release()

	var row idRow

	conn.QueryRow(context.Background(), `INSERT INTO characters 
	(name, class, level, aa, is_bot, created_by) 
	VALUES ($1, $2, $3, $4, $5, $6) RETURNING id;`,
		r.Name,
		r.Class,
		r.Level,
		r.AA,
		r.Bot,
		r.CreatedBy,
	).Scan(&row)

	r.Id = row.Id

	return nil
}
