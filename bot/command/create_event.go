package command

import (
	"eqRaidBot/db/model"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	eventStateStart     = 0
	eventStateName      = 1
	eventStateDesc      = 2
	eventStateTime      = 3
	eventStateRepeating = 4
	eventStateDone      = 5
	eventStateSaved     = 6
)

type CreateEventProvider struct {
	pool     *pgxpool.Pool
	registry map[string]eventState
	manifest *Manifest
}

type eventState struct {
	userId      string
	name        string
	description string
	time        time.Time
	repeats     bool
	state       int64
}

func (r *eventState) IsComplete() bool {
	return r.state == eventStateSaved
}

func (r *eventState) toModel() *model.Event {
	return &model.Event{
		Title:        r.name,
		Description:  r.description,
		EventTime:    r.time,
		IsRepeatable: r.repeats,
		CreatedBy:    r.userId,
	}
}

func NewCreateEventProvider(db *pgxpool.Pool) *CreateEventProvider {
	provider := &CreateEventProvider{
		pool:     db,
		registry: make(map[string]eventState),
	}

	steps := []Step{
		provider.start,
		provider.name,
		provider.description,
		provider.time,
		provider.repeating,
		provider.done,
	}

	provider.manifest = &Manifest{Steps: steps}

	return provider
}

func (r *CreateEventProvider) Name() string {
	return "!event-create"
}

func (r *CreateEventProvider) Description() string {
	return "begins an event creation workflow, may not be available to all users"
}

func (r *CreateEventProvider) Cleanup() {
}

func (r *CreateEventProvider) WorkflowForUser(userId string) State {
	if v, ok := r.registry[userId]; ok {
		return &v
	} else {
		return nil
	}
}

func (r *CreateEventProvider) Handle(s *discordgo.Session, m *discordgo.MessageCreate) {
	c, err := s.UserChannelCreate(m.Author.ID)
	if err != nil {
		log.Print(err.Error())
		return
	}

	action, err := processCommand(r.manifest, 0, m, s, c.ID)
	if action == actionSent {
		return
	}

	if _, ok := r.registry[m.Author.ID]; !ok {
		err = sendMessage(s, c.ID, "Please restart the event creation process by typing **!create-event**")
		if err != nil {
			return
		}
	}

	reg := r.registry[m.Author.ID]

	_, err = processCommand(r.manifest, reg.state, m, s, c.ID)
	if err != nil {
		log.Println(err.Error())
	}
}

func (r *CreateEventProvider) start(m *discordgo.MessageCreate) (string, error) {
	if _, ok := r.registry[m.Author.ID]; !ok {
		r.registry[m.Author.ID] = eventState{
			state:  eventStateName,
			userId: m.Author.ID,
		}
		return fmt.Sprintf("Hello %s, what should we call this event?", m.Author.Username), nil
	}
	return "", nil
}

func (r *CreateEventProvider) name(m *discordgo.MessageCreate) (string, error) {
	v := r.registry[m.Author.ID]
	v.name = m.Content
	v.state = eventStateDesc
	r.registry[m.Author.ID] = v

	return "Enter a description", nil
}

func (r *CreateEventProvider) description(m *discordgo.MessageCreate) (string, error) {
	v := r.registry[m.Author.ID]
	v.description = m.Content
	v.state = eventStateTime
	r.registry[m.Author.ID] = v

	return `Enter a time for the event.  
Time must be in the following format: **01/21/2022 07:00PM EST**`, nil
}

func (r *CreateEventProvider) time(m *discordgo.MessageCreate) (string, error) {
	v := r.registry[m.Author.ID]

	t, err := time.Parse("01/02/2006 03:04PM MST", m.Content)
	if err != nil {
		return "", ErrorInvalidInput
	}
	v.time = t.UTC()
	v.state = eventStateRepeating
	r.registry[m.Author.ID] = v

	return `Does the event repeat?. (1 or 2) 
1. Yes
2. No`, nil
}

func (r *CreateEventProvider) repeating(m *discordgo.MessageCreate) (string, error) {
	v := r.registry[m.Author.ID]

	switch m.Message.Content {
	case "1":
		v.repeats = true
	case "2":
		v.repeats = false
	default:
		return "", ErrorInvalidInput
	}

	v.state = eventStateDone
	r.registry[m.Author.ID] = v

	msg := `Does this all look correct?. (1 or 2) 
Title: %s
Description: %s
Time: %s
Repeating: %t

1. Yes
2. No`

	return fmt.Sprintf(msg, v.name, v.description, v.time.String(), v.repeats), nil

}

func (r *CreateEventProvider) done(m *discordgo.MessageCreate) (string, error) {
	if m.Content == "1" {
		dat := r.registry[m.Author.ID]
		err := dat.toModel().Save(r.pool)
		if err != nil {
			log.Printf(err.Error())
			return "", ErrorInternalError
		}
		r.reset(m)
		return "The event has been saved", nil
	} else if m.Content == "2" {
		r.reset(m)
		return "Resetting the event", nil
	} else {
		return "", ErrorInvalidInput
	}
}

func (r *CreateEventProvider) reset(m *discordgo.MessageCreate) {
	delete(r.registry, m.Author.ID)
}
