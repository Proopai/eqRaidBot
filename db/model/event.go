package model

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

type Event struct {
	Id           int64
	Title        string
	Description  string
	EventTime    time.Time
	IsRepeatable bool
	CreatedBy    string
	CreatedAt    time.Time
}

type idRow struct {
	Id int64
}

func (r *Event) Save(db *pgxpool.Pool) error {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return err
	}

	defer conn.Release()

	var row idRow

	conn.QueryRow(context.Background(), `INSERT INTO events 
	(title, description, event_time, is_repeatable, created_by) 
	VALUES ($1, $2, $3, $4, $5) RETURNING id;`,
		r.Title,
		r.Description,
		r.EventTime,
		r.IsRepeatable,
		r.CreatedBy,
	).Scan(&row)

	r.Id = row.Id

	return nil
}
