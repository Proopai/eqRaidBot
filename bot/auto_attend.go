package bot

import (
	"eqRaidBot/db/model"
	"github.com/jackc/pgx/v4/pgxpool"
	"log"
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
			log.Println("Stopping auto-attender...")
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

	now := time.Now()

	events, err := eventProvider.GetAll(a.db)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		log.Printf("No events found")
		return nil
	}

	log.Printf("processing %d events...", len(events))

	for _, event := range events {
		toons, err := characterProvider.GetAllNotAttendingEvent(a.db, event.Id)
		if err != nil {
			return err
		}

		if len(toons) == 0 {
			continue
		}

		var attendance []model.Attendance
		for _, v := range toons {
			attendance = append(attendance, model.Attendance{
				EventId:     event.Id,
				CharacterId: v.Id,
				Withdrawn:   false,
			})
		}

		log.Printf("Saving %d members for event %d", len(attendance), event.Id)
		err = attendanceProvider.SaveBatch(a.db, attendance)
		if err != nil {
			return err
		}
	}

	log.Printf("done in %f...", time.Since(now).Seconds())
	return nil
}
