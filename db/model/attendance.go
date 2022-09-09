package model

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Attendance struct {
	EventId     int64
	CharacterId int64
	Withdrawn   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (r *Attendance) Save(db *pgxpool.Pool) error {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return err
	}

	defer conn.Release()

	conn.QueryRow(context.Background(), `INSERT INTO attendance 
	(character_id, event_id, withdrawn, updated_at) 
	VALUES ($1, $2, $3, NOW());`,
		r.CharacterId,
		r.EventId,
		r.Withdrawn,
	)

	return nil
}

func (r *Attendance) Update(db *pgxpool.Pool) error {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return err
	}

	defer conn.Release()

	conn.QueryRow(context.Background(), `UPDATE attendance 
SET withdrawn=$1, updated_at=NOW() 
WHERE event_id=$2 AND character_id=$3;`,
		r.Withdrawn,
		r.EventId,
		r.CharacterId,
	)

	return nil
}

func (r *Attendance) SaveBatch(db *pgxpool.Pool, rows []Attendance) error {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return err
	}

	defer conn.Release()

	var (
		params []string
		vals   []interface{}
	)

	now := time.Now()

	for _, r := range rows {
		t := len(vals)
		params = append(params, fmt.Sprintf("($%d, $%d, $%d, $%d)", t+1, t+2, t+3, t+4))
		vals = append(vals, r.CharacterId)
		vals = append(vals, r.EventId)
		vals = append(vals, r.Withdrawn)
		vals = append(vals, now)
	}

	q := fmt.Sprintf("INSERT INTO attendance (character_id, event_id, withdrawn, updated_at) VALUES %s;", strings.Join(params, ","))
	conn.QueryRow(context.Background(), q, vals...)

	return nil

}

func (r *Attendance) GetAttendees(db *pgxpool.Pool, eventId int64) ([]Character, error) {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return nil, err
	}

	var attendees []Character
	pgxscan.Select(context.Background(), db, &attendees, `SELECT * from characters where id IN (SELECT character_id FROM attendance 
	WHERE event_id = $1 and withdrawn = false);`, eventId)

	defer conn.Release()

	return attendees, nil
}

func (r *Attendance) GetMyAttendanceForEvent(db *pgxpool.Pool, eventId int64, userId string) ([]Attendance, error) {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return nil, err
	}

	var attendees []Attendance
	pgxscan.Select(context.Background(), db, &attendees, `SELECT a.* from attendance a
LEFT JOIN characters c on a.character_id = c.id 
WHERE c.created_by=$1
AND a.event_id=$2;`, userId, eventId)

	defer conn.Release()

	return attendees, nil
}

func (r *Attendance) GetPendingAttendance(db *pgxpool.Pool, userId string) ([]Attendance, error) {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return nil, err
	}

	var attendees []Attendance
	pgxscan.Select(context.Background(), db, &attendees, `SELECT a.* from attendance a
LEFT JOIN characters c on a.character_id = c.id 
LEFT JOIN events e on a.event_id = e.id
WHERE c.created_by=$1
AND a.withdrawn=false
AND e.event_time > NOW();`, userId)

	defer conn.Release()

	return attendees, nil
}

func (r *Attendance) GetAttendeesForEvents(db *pgxpool.Pool, eventIds []int64) (map[int64][]Character, error) {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return nil, err
	}

	var (
		idStrs []string
		ids    []interface{}
	)
	for k, v := range eventIds {
		idStrs = append(idStrs, fmt.Sprintf("$%d", k+1))
		ids = append(ids, v)
	}

	var attendees []Attendance
	err = pgxscan.Select(context.Background(), db, &attendees, fmt.Sprintf(`SELECT * FROM attendance WHERE event_id IN (%s) AND withdrawn=false;`, strings.Join(idStrs, ", ")), ids...)
	if err != nil {
		log.Print(err.Error())
	}

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
