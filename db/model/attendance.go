package model

import "time"

type Attendance struct {
	EventId     int64
	CharacterId int64
	IsWithdrawn int64
	CreatedAt   time.Time
}
