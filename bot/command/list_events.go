package command

import (
	"eqRaidBot/db/model"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
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
	return ListEvents
}

func (r *ListEventProvider) Description() string {
	return "lists all events that have not yet begun"
}

func (r *ListEventProvider) Cleanup() {
}

func (r *ListEventProvider) Reset(m *discordgo.MessageCreate) {
}

func (r *ListEventProvider) WorkflowForUser(userId string) State {
	return nil
}

func (r *ListEventProvider) Handle(s *discordgo.Session, m *discordgo.MessageCreate) {
	genericSimpleHandler(s, m, r.manifest)
}

func (r *ListEventProvider) list(m *discordgo.MessageCreate) (string, error) {
	var eventListText = `All scheduled events are listed below.
%s
`
	e := model.Event{}
	rows, err := e.GetAll(r.pool)
	if err != nil {
		return "", ErrorInternalError
	}

	var eventIds []int64
	for _, event := range rows {
		eventIds = append(eventIds, event.Id)
	}
	at := model.Attendance{}
	attendeeMap, err := at.GetAttendeesForEvents(r.pool, eventIds)
	if err != nil {
		return "", ErrorInternalError
	}

	var eventList []string
	for i, r := range rows {
		eventList = append(eventList, fmt.Sprintf("**%d. %s %s**: %s (%d)", i+1, r.EventTime.Format(time.RFC822), r.Title, r.Description, len(attendeeMap[r.Id])))
	}

	return fmt.Sprintf(eventListText, strings.Join(eventList, "\n")), nil
}
