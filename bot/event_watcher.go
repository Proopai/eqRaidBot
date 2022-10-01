package bot

import (
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

	return nil
}
