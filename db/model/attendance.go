package model

import (
	"context"
	"time"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Attendance struct {
	EventId     int64
	CharacterId int64
	IsWithdrawn int64
	CreatedAt   time.Time
}

func (r *Attendance) GetByOwner(db *pgxpool.Pool, userId string) ([]Character, error) {
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
