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
	(character_id, event_id, withdrawn, updated_at) 
	VALUES ($1, $2, $3, $4);`,
		r.CharacterId,
		r.EventId,
		r.IsWithdrawn,
		time.Now(),
	)

	return nil
}

func (r *Attendance) GetAttendees(db *pgxpool.Pool, eventId int64) ([]Character, error) {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return nil, err
	}

	var attendees []Attendance
	pgxscan.Select(context.Background(), db, &attendees, `SELECT * FROM attendance 
	WHERE event_id = $1;`, eventId)

	defer conn.Release()

	var charIds []int64
	for _, i := range attendees {
		charIds = append(charIds, i.CharacterId)
	}

	char := Character{}
	toons, err := char.GetWhereIn(db, charIds)
	if err != nil {
		return nil, err
	}

	return toons, nil

}

func (r *Attendance) GetAttendeesForEvents(db *pgxpool.Pool, eventIds []int64) (map[int64][]Character, error) {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return nil, err
	}

	var attendees []Attendance
	pgxscan.Select(context.Background(), db, &attendees, `SELECT * FROM attendance 
	WHERE event_id IN ($1);`, eventIds)

	defer conn.Release()

	var charIds []int64
	for _, i := range attendees {
		charIds = append(charIds, i.CharacterId)
	}

	char := Character{}
	toons, err := char.GetWhereIn(db, charIds)
	if err != nil {
		return nil, err
	}

	cMap := make(map[int64]Character)
	for _, c := range toons {
		cMap[c.Id] = c
	}

	res := make(map[int64][]Character)
	for _, a := range attendees {
		if _, ok := res[a.EventId]; !ok {
			res[a.EventId] = []Character{cMap[a.CharacterId]}
			continue
		}
		res[a.EventId] = append(res[a.EventId], cMap[a.CharacterId])
	}

	return res, nil

}
