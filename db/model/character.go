package model

import "time"

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

func (r *Character) Save() error {
	return nil
}
