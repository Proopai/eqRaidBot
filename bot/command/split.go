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
	}
}

func (r *SplitProvider) Split(s *discordgo.Session, m *discordgo.MessageCreate) {
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
		err := r.ackEvent(c, s, m)
		if err != nil {
			_ = sendMessage(s, c, "There was a problem with this request")
		}
	case splitStateEvent:
		err := r.ackSplit(c, s, m)
		if err != nil {
			_ = sendMessage(s, c, "There was a problem with this request")
		}
	case splitStateSplit:
	}

}

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
		vs.state = splitStateSplit
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
	splitter := eq.NewSplitter(attendees)

	splitter.Split(i)

	return nil
}

func (r *SplitProvider) init(c *discordgo.Channel, s *discordgo.Session, m *discordgo.MessageCreate) (bool, error) {
	// check the database to see if they have previously registered
	if _, ok := r.registry[m.Author.ID]; !ok {
		e := model.Event{}
		events, err := e.GetAll(r.pool)
		if err != nil {
			log.Println("error")
			_ = sendMessage(s, c, "There was a problem with this request")
			return false, err
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
