package model

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	err = pgxscan.Select(context.Background(), db, &events, `SELECT * FROM events 
	WHERE event_time > current_timestamp order by event_time;`)
	fmt.Println(events, err)
	if err != nil {
		return nil, err
	}

	fmt.Println(events, err)

	return events, nil
}

func (r *Event) GetNext(db *pgxpool.Pool) (Event, error) {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return Event{}, err
	}

	defer conn.Release()

	var events []Event
	pgxscan.Select(context.Background(), db, &events, `SELECT * FROM events 
	WHERE event_time > NOW() order by event_time limit 1;`)

	if len(events) > 0 {
		return events[0], nil
	}

	return Event{}, errors.New("not found")
}

func (r *Event) GetWhereIn(db *pgxpool.Pool, eventIds []int64) ([]Event, error) {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return nil, err
	}

	defer conn.Release()

	var events []Event
	var part []string
	seen := make(map[int64]bool)
	for _, id := range eventIds {
		if _, ok := seen[id]; ok {
			continue
		}
		part = append(part, fmt.Sprintf("%d", id))
		seen[id] = true
	}
	pgxscan.Select(context.Background(), db, &events, fmt.Sprintf(`SELECT * FROM events 
	WHERE id IN (%s);`, strings.Join(part, ",")))

	return events, nil
}
