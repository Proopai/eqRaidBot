package model

import (
	"context"
	"time"

	"github.com/georgysavva/scany/pgxscan"
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

func (r *Event) GetAll(db *pgxpool.Pool) ([]Event, error) {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return nil, err
	}

	defer conn.Release()

	var events []Event
	pgxscan.Select(context.Background(), db, &events, `SELECT * FROM events 
	WHERE event_time > NOW() order by event_time desc;`)

	return events, nil
}
