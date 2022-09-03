package command

import (
	"eqRaidBot/bot/eq"
	"eqRaidBot/db/model"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	splitStateStart = 1
	splitStateEvent = 2
	splitStateSplit = 3
	splitStateDone  = 4
)

type SplitProvider struct {
	pool     *pgxpool.Pool
	registry map[string]SplitState
	eventReg map[string]map[int]model.Event
}

type SplitState struct {
	eventId int64
	state   int
	userId  string
}

func NewSplitProvider(db *pgxpool.Pool) *SplitProvider {
	return &SplitProvider{
		pool:     db,
		eventReg: make(map[string]map[int]model.Event),
		registry: make(map[string]SplitState),
	}
}

func (r *SplitState) IsComplete() bool {
	return r.state == splitStateSplit && r.eventId != 0
}

func (r *SplitProvider) SplitWorkflow(userId string) *SplitState {
	if v, ok := r.registry[userId]; ok {
		return &v
	} else {
		return nil
	}
}

func (r *SplitProvider) Step(s *discordgo.Session, m *discordgo.MessageCreate) {
	c, err := s.UserChannelCreate(m.Author.ID)
	if err != nil {
		log.Print(err.Error())
		return
	}

	actioned, err := r.init(c, s, m)
	if err != nil {
		_ = sendMessage(s, c, err.Error())
		return
	}

	if actioned {
		return
	}

	switch r.registry[m.Author.ID].state {
	case splitStateStart:
		if err := r.ackEvent(c, s, m); err != nil {
			log.Println(err.Error())
			_ = sendMessage(s, c, "There was a problem with this request")
		}
	case splitStateEvent:
		if err := r.ackSplit(c, s, m); err != nil {
			log.Println(err.Error())
			_ = sendMessage(s, c, "There was a problem with this request")
		}
		// case splitStateSplit:
		// 	if err := r.ackDone(c, s, m); err != nil {
		// 		_ = sendMessage(s, c, "There was a problem with this request")
		// 	}
	}

}

// func (r *SplitProvider) ackDone(c *discordgo.Channel, s *discordgo.Session, m *discordgo.MessageCreate) error {
// }

func (r *SplitProvider) ackEvent(c *discordgo.Channel, s *discordgo.Session, m *discordgo.MessageCreate) error {
	i, err := strconv.Atoi(m.Content)
	if err != nil {
		return err
	}

	vs := r.registry[m.Author.ID]

	for k, v := range r.eventReg[m.Author.ID] {
		if k == i {
			vs.eventId = v.Id
			break
		}
	}

	if vs.eventId == 0 {
		return errors.New("invalid event selection")
	} else {
		vs.state = splitStateEvent
		r.registry[m.Author.ID] = vs
	}

	_ = sendMessage(s, c, "How many ways should I split this event? e.g. 4")

	return nil
}

func (r *SplitProvider) ackSplit(c *discordgo.Channel, s *discordgo.Session, m *discordgo.MessageCreate) error {
	i, err := strconv.Atoi(m.Content)
	if err != nil {
		return err
	}

	if i < 2 {
		return errors.New("You cannot one split an event.")
	}

	a := model.Attendance{}

	attendees, err := a.GetAttendees(r.pool, r.registry[m.Author.ID].eventId)
	if err != nil {
		return err
	}

	var splitString string

	splitter := eq.NewSplitter(attendees, false)
	splits := splitter.Split(i)

	for i, split := range splits {
		splitString += fmt.Sprintf("**Raid %d\n**", i)
		for g, group := range split {
			splitString += fmt.Sprintf("Group %d\n", g)
			for j, c := range group {
				if j == 0 {
					splitString += fmt.Sprintf("**%s - %s**\n", eq.ClassChoiceMap[c.Class], c.Name)
				} else {

					splitString += fmt.Sprintf("%s - %s\n", eq.ClassChoiceMap[c.Class], c.Name)
				}
			}
		}
	}

	return sendMessage(s, c, splitString)
}

func (r *SplitProvider) init(c *discordgo.Channel, s *discordgo.Session, m *discordgo.MessageCreate) (bool, error) {
	// check the database to see if they have previously registered
	if _, ok := r.registry[m.Author.ID]; !ok {
		e := model.Event{}
		events, err := e.GetAll(r.pool)
		if err != nil {
			return false, err
		}

		if len(events) == 0 {
			return false, errors.New("There are no events.")
		}

		r.registry[m.Author.ID] = SplitState{
			state:  splitStateStart,
			userId: m.Author.ID,
		}
		r.eventReg[m.Author.ID] = make(map[int]model.Event)

		var eventString []string
		for i, e := range events {
			r.eventReg[m.Author.ID][i] = e
			eventString = append(eventString, fmt.Sprintf("%d. %s %s", i, e.Title, e.EventTime.Format(time.RFC822)))
		}

		_ = sendMessage(s, c, "What event would you like to split?\n"+strings.Join(eventString, "\n"))
		return true, nil
	}

	return false, nil
}
