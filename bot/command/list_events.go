package command

import (
	"eqRaidBot/db/model"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
	"log"
	"strings"
	"time"
)

type ListEventProvider struct {
	pool     *pgxpool.Pool
	manifest *Manifest
}

func NewListEventsProvider(db *pgxpool.Pool) *ListEventProvider {
	provider := &ListEventProvider{pool: db}

	steps := []Step{
		provider.list,
	}

	provider.manifest = &Manifest{Steps: steps}

	return provider
}

func (r *ListEventProvider) Name() string {
	return "!list-events"
}

func (r *ListEventProvider) Description() string {
	return "lists all events that have not passed and have been created"
}

func (r *ListEventProvider) Cleanup() {
}

func (r *ListEventProvider) WorkflowForUser(userId string) State {
	return nil
}

func (r *ListEventProvider) Handle(s *discordgo.Session, m *discordgo.MessageCreate) {
	if _, err := processCommand(r.manifest, 0, m, s, m.ChannelID); err != nil {
		log.Println(err.Error())
	}
}

func (r *ListEventProvider) list(m *discordgo.MessageCreate) (string, error) {
	var eventListText = `All scheduled events are listed below.
%s
`
	e := model.Event{}
	rows, err := e.GetAll(r.pool)
	if err != nil {
		return "", errors.New("there was a problem with this request")
	}

	var eventIds []int64
	for _, event := range rows {
		eventIds = append(eventIds, event.Id)
	}
	at := model.Attendance{}
	attendeeMap, err := at.GetAttendeesForEvents(r.pool, eventIds)
	if err != nil {
		return "", errors.New("there was a problem with this request")
	}

	var eventList []string
	for i, r := range rows {
		eventList = append(eventList, fmt.Sprintf("**%d. %s %s**: %s (%d)", i+1, r.EventTime.Format(time.RFC822), r.Title, r.Description, len(attendeeMap[r.Id])))
	}

	return fmt.Sprintf(eventListText, strings.Join(eventList, "\n")), nil
}
