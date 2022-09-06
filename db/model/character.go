package model

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Character struct {
	Id            int64
	Name          string
	Class         int64
	Level         int64
	AA            int64
	CharacterType int64
	CreatedBy     string
	CreatedAt     time.Time
}

func (r *Character) Save(db *pgxpool.Pool) error {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return err
	}

	defer conn.Release()

	var row idRow

	conn.QueryRow(context.Background(), `INSERT INTO characters 
	(name, class, level, aa, character_type, created_by) 
	VALUES ($1, $2, $3, $4, $5, $6) RETURNING id;`,
		r.Name,
		r.Class,
		r.Level,
		r.AA,
		r.CharacterType,
		r.CreatedBy,
	).Scan(&row)

	r.Id = row.Id

	return nil
}

func (r *Character) GetByOwner(db *pgxpool.Pool, userId string) ([]Character, error) {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return nil, err
	}

	defer conn.Release()

	var toons []Character
	q := `SELECT * FROM characters 
	WHERE created_by = $1 order by level desc;`
	if err = pgxscan.Select(context.Background(), db, &toons, q, userId); err != nil {
		return nil, err
	}

	return toons, nil
}

func (r *Character) GetAllActive(db *pgxpool.Pool) ([]Character, error) {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	var toons []Character
	// types main and box
	q := `SELECT * FROM characters where character_type IN(1,2) order by level desc;`
	if err = pgxscan.Select(context.Background(), db, &toons, q); err != nil {
		return nil, err
	}

	return toons, nil
}

func (r *Character) GetAllNotAttendingEvent(db *pgxpool.Pool, eventId int64) ([]Character, error) {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	var toons []Character
	// types main and box
	q := `SELECT * FROM characters 
where character_type IN(1,2) 
and id NOT IN (select character_id from attendance where event_id = $1)
order by level desc;`
	if err = pgxscan.Select(context.Background(), db, &toons, q, eventId); err != nil {
		return nil, err
	}

	return toons, nil
}

func (r *Character) GetAllAttendingEvent(db *pgxpool.Pool, eventId int64) ([]Character, error) {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	var toons []Character
	// types main and box
	q := `SELECT * FROM characters 
where character_type IN(1,2) 
and id IN (select character_id from attendance where event_id = $1)
order by level desc;`
	if err = pgxscan.Select(context.Background(), db, &toons, q, eventId); err != nil {
		return nil, err
	}

	return toons, nil
}

func (r *Character) GetWhereIn(db *pgxpool.Pool, characterIds []int64) ([]Character, error) {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return nil, err
	}

	defer conn.Release()

	var toons []Character
	var part []string
	seen := make(map[int64]bool)
	for _, id := range characterIds {
		if _, ok := seen[id]; ok {
			continue
		}
		part = append(part, fmt.Sprintf("%d", id))
		seen[id] = true
	}

	pgxscan.Select(context.Background(), db, &toons, fmt.Sprintf(`SELECT * FROM characters 
	WHERE id IN (%s);`, strings.Join(part, ", ")))

	return toons, nil
}
