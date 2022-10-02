package bot

import (
	"eqRaidBot/db/model"
	"github.com/jackc/pgx/v4/pgxpool"
	"log"
	"time"
)

type EventWatcher struct {
	db *pgxpool.Pool
}

func NewEventWatcher(db *pgxpool.Pool) *EventWatcher {
	return &EventWatcher{db: db}
}

func (a *EventWatcher) Run(stop <-chan struct{}, d time.Duration) {
	t := time.NewTicker(d)
	for {
		select {
		case <-stop:
			t.Stop()
			log.Println("Stopping event watcher...")
			return
		case <-t.C:
			err := a.checkEvents()
			if err != nil {
				log.Printf(err.Error())
			}
		}
	}
}

func (a *EventWatcher) checkEvents() error {
	eventProvider := model.Event{}
	events, err := eventProvider.GetAllNeedsRenewal(a.db)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		return nil
	}

	seen := make(map[string]bool)

	for _, e := range events {
		if a.needsRenewal(e) {
			if _, ok := seen[e.Title]; !ok {
				event := model.Event{
					Title:        e.Title,
					Description:  e.Description,
					EventTime:    e.EventTime.Add(24 * 7 * time.Hour),
					IsRepeatable: true,
					CreatedBy:    e.CreatedBy,
				}

				err = event.Save(a.db)
				if err != nil {
					continue
				}

				seen[e.Title] = true
			}
		}
	}
	return nil
}

func (a *EventWatcher) needsRenewal(e model.Event) bool {
	return e.EventTime.Before(time.Now()) && e.IsRepeatable
}
