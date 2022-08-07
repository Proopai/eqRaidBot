package bot

import (
	"eqRaidBot/db/model"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	eventStateName      = 0
	eventStateDesc      = 1
	eventStateTime      = 2
	eventStateRepeating = 3
	eventStateDone      = 4
	eventStateSaved     = 5
)

type EventProvider struct {
	pool     *pgxpool.Pool
	registry map[string]eventState
}

type eventState struct {
	userId      string
	name        string
	description string
	time        time.Time
	repeats     bool
	state       int64
}

func (r *eventState) isComplete() bool {
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

func NewEventProvider(db *pgxpool.Pool) *EventProvider {
	return &EventProvider{
		pool:     db,
		registry: make(map[string]eventState),
	}
}

var eventListText = `All scheduled events are listed below.
%s
`

func (r *EventProvider) listEvents(s *discordgo.Session, m *discordgo.MessageCreate) {
	_, err := s.ChannelMessageSend(m.ChannelID, eventListText)
	if err != nil {
		log.Println(err.Error())
	}
}

func (r *EventProvider) eventWorkflow(userId string) *eventState {
	if v, ok := r.registry[userId]; ok {
		return &v
	} else {
		return nil
	}
}

func (r *EventProvider) createEventStep(s *discordgo.Session, m *discordgo.MessageCreate) {
	c, err := s.UserChannelCreate(m.Author.ID)
	if err != nil {
		log.Print(err.Error())
		return
	}

	if actioned := r.init(c, s, m); actioned {
		return
	}

	e := r.registry[m.Author.ID]

	switch e.state {
	case eventStateName:
		r.nameAck(m)
		if err := sendMessage(s, c, "Enter a description"); err != nil {
			log.Print(err.Error())
		}
	case eventStateDesc:
		msg := `Enter a time for the event.  
Time must be in the following format: **01/21/2022 07:00PM PST**`
		r.descriptionAck(m)
		if err := sendMessage(s, c, msg); err != nil {
			log.Print(err.Error())
		}
	case eventStateTime:
		if err := r.timeAck(m); err != nil {
			if err := sendMessage(s, c, "There was a problem parsing your time input, try again."); err != nil {
				log.Print(err.Error())
			}
			return
		}

		msg := `Does the event repeat?. (1 or 2) 
1. Yes
2. No`

		if err := sendMessage(s, c, msg); err != nil {
			log.Print(err.Error())
		}
	case eventStateRepeating:
		if err := r.repeatingAck(m); err != nil {
			if err := sendMessage(s, c, "There was a problem parsing your time input, try again."); err != nil {
				log.Print(err.Error())
			}
			return
		}

		msg := `Does this all look correct?. (1 or 2) 
Title: %s
Description: %s
Time: %s
Repeating: %t

1. Yes
2. No`

		if err := sendMessage(s, c, fmt.Sprintf(msg, e.name, e.description, e.time.String(), r.registry[m.Author.ID].repeats)); err != nil {
			log.Print(err.Error())
		}
	case eventStateDone:
		if m.Content == "1" {
			dat := r.registry[m.Author.ID]
			err := dat.toModel().Save(r.pool)

			if err != nil {
				log.Printf(err.Error())
				_ = sendMessage(s, c, "There was an error saving the event!")
				return
			}

			if err = sendMessage(s, c, "The event has been saved."); err != nil {
				log.Println(err.Error())
			}

			r.reset(m)
		} else if m.Content == "2" {
			if err = sendMessage(s, c, "Resetting the event."); err != nil {
				log.Println(err.Error())
			}

			r.reset(m)
			r.init(c, s, m)
		} else {
			_ = sendMessage(s, c, "There was an error with your input - please try again")
		}
	}

}

func (r *EventProvider) init(c *discordgo.Channel, s *discordgo.Session, m *discordgo.MessageCreate) bool {
	if _, ok := r.registry[m.Author.ID]; !ok {
		r.registry[m.Author.ID] = eventState{
			state:  eventStateName,
			userId: m.Author.ID,
		}

		if err := sendMessage(s, c, fmt.Sprintf("Hello %s, what should we call this event?", m.Author.Username)); err != nil {
			return false
		}

		return true
	}

	return false
}

func (r *EventProvider) reset(m *discordgo.MessageCreate) {
	delete(r.registry, m.Author.ID)
}

func (r *EventProvider) nameAck(m *discordgo.MessageCreate) {
	v := r.registry[m.Author.ID]
	v.name = m.Content
	v.state = eventStateDesc
	r.registry[m.Author.ID] = v
}

func (r *EventProvider) descriptionAck(m *discordgo.MessageCreate) {
	v := r.registry[m.Author.ID]
	v.description = m.Content
	v.state = eventStateTime
	r.registry[m.Author.ID] = v
}

func (r *EventProvider) timeAck(m *discordgo.MessageCreate) error {
	v := r.registry[m.Author.ID]

	t, err := time.Parse("01/02/2006 03:04PM MST", m.Content)
	if err != nil {
		return err
	}

	v.time = t
	v.state = eventStateRepeating
	r.registry[m.Author.ID] = v

	return nil
}

func (r *EventProvider) repeatingAck(m *discordgo.MessageCreate) error {
	v := r.registry[m.Author.ID]

	switch m.Message.Content {
	case "1":
		v.repeats = true
	case "2":
		v.repeats = false
	default:
		return errors.New("invalid input")
	}

	v.state = eventStateDone
	r.registry[m.Author.ID] = v

	return nil
}
