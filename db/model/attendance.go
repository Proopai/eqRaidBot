package model

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

type Attendance struct {
	EventId     int64
	CharacterId int64
	IsWithdrawn bool
	CreatedAt   time.Time
}

func (r *Attendance) Save(db *pgxpool.Pool) error {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return err
	}

	defer conn.Release()

	conn.QueryRow(context.Background(), `INSERT INTO attendance 
	(character_id, event_id, withdrawn) 
	VALUES ($1, $2, $3);`,
		r.CharacterId,
		r.EventId,
		r.IsWithdrawn,
	)

	return nil
}
