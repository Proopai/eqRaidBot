package model

import "time"

type Event struct {
	Id           int64
	Title        string
	Description  string
	EventTime    time.Time
	IsRepeatable bool
	CreatedBy    string
	CreatedAt    time.Time
}
