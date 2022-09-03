package bot

import (
	"eqRaidBot/db/model"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog/log"
	"time"
)

type AutoAttender struct {
	db *pgxpool.Pool
}

func NewAutoAttender(db *pgxpool.Pool) *AutoAttender {
	return &AutoAttender{
		db: db,
	}
}

func (a *AutoAttender) Run(stop <-chan struct{}, d time.Duration) {
	t := time.NewTicker(d)
	for {
		select {
		case <-stop:
			t.Stop()
			// shut down
			return
		case <-t.C:
			err := a.registerMembers()
			if err != nil {
				log.Printf(err.Error())
			}
		}
	}
}

func (a *AutoAttender) registerMembers() error {
	eventProvider := model.Event{}
	characterProvider := model.Character{}
	attendanceProvider := model.Attendance{}

	events, err := eventProvider.GetAll(a.db)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		log.Printf("No events found")
		return nil
	}

	for _, event := range events {
		toons, err := characterProvider.GetAllNotAttendingEvent(a.db, event.Id)
		if err != nil {
			return err
		}

		if len(toons) == 0 {
			log.Printf("No toons found for event %d", event.Id)
			continue
		}

		var attendance []model.Attendance
		for _, v := range toons {
			attendance = append(attendance, model.Attendance{
				EventId:     event.Id,
				CharacterId: v.Id,
				IsWithdrawn: false,
			})
		}

		log.Printf("Saving %d members for event %d", len(attendance), event.Id)
		err = attendanceProvider.SaveBatch(a.db, attendance)
		if err != nil {
			return err
		}
	}
	return nil
}
