package command

import (
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
	attendStateChar  = 1
	attendStateEvent = 2
	attendStateDone  = 3
	attendStateSaved = 4
)

type AttendanceProvider struct {
	pool     *pgxpool.Pool
	registry map[string]AttendanceState
	charReg  map[string]map[int]model.Character
	eventReg map[string]map[int]model.Event
}

type AttendanceState struct {
	characterId int64
	eventId     int64
	state       int
	userId      string
}

func (r *AttendanceState) toModel() *model.Attendance {
	return &model.Attendance{
		EventId:     r.eventId,
		CharacterId: r.characterId,
		IsWithdrawn: false,
	}
}

func (r *AttendanceState) IsComplete() bool {
	return r.state == attendStateSaved && r.characterId != 0 && r.eventId != 0
}

func NewAttendanceProvider(db *pgxpool.Pool) *AttendanceProvider {
	return &AttendanceProvider{
		registry: make(map[string]AttendanceState),
		charReg:  make(map[string]map[int]model.Character),
		eventReg: make(map[string]map[int]model.Event),
		pool:     db,
	}
}

func (r *AttendanceProvider) AttendWorkflow(userId string) *AttendanceState {
	if v, ok := r.registry[userId]; ok {
		return &v
	} else {
		return nil
	}
}

func (r *AttendanceProvider) Step(s *discordgo.Session, m *discordgo.MessageCreate) {
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
	case attendStateChar:
		err := r.charAck(c, s, m)
		if err != nil {
			log.Println(err.Error())
			_ = sendMessage(s, c, "There was a problem fetching the events")
		}
	case attendStateEvent:
		err := r.eventAck(c, s, m)
		if err != nil {
			_ = sendMessage(s, c, "There was a problem fetching the events")
		}
	case attendStateDone:
		if err := r.doneAck(s, c, m); err != nil {
			_ = sendMessage(s, c, "There was an error with your input - please try again")
		}
	}

}

func (r *AttendanceProvider) init(c *discordgo.Channel, s *discordgo.Session, m *discordgo.MessageCreate) (bool, error) {
	// check the database to see if they have previously registered
	if _, ok := r.registry[m.Author.ID]; !ok {
		char := model.Character{}
		toons, err := char.GetByOwner(r.pool, m.Author.ID)
		if err != nil {
			log.Println(err.Error())
			return false, errors.New("There was an error with your input - please try again")
		}

		if len(toons) == 0 {
			return false, errors.New("You have no characters register, please type **!register** to add one.")
		}

		r.registry[m.Author.ID] = AttendanceState{
			state:  attendStateChar,
			userId: m.Author.ID,
		}

		r.charReg[m.Author.ID] = make(map[int]model.Character)
		r.eventReg[m.Author.ID] = make(map[int]model.Event)

		var charString []string
		for i, t := range toons {
			r.charReg[m.Author.ID][i] = t
			charString = append(charString, fmt.Sprintf("%d. %s", i, t.Name))
		}

		if err := sendMessage(s, c, fmt.Sprintf("Hello %s, which character will you be brining?\n%s", m.Author.Username, strings.Join(charString, "\n"))); err != nil {
			return true, err
		}
		return true, nil
	}

	return false, nil
}

func (r *AttendanceProvider) charAck(c *discordgo.Channel, s *discordgo.Session, m *discordgo.MessageCreate) error {
	i, err := strconv.Atoi(m.Content)
	if err != nil {
		return err
	}

	vs := r.registry[m.Author.ID]

	for k, v := range r.charReg[m.Author.ID] {
		if k == i {
			vs.characterId = v.Id
			break
		}
	}

	if vs.characterId == 0 {
		return errors.New("invalid character selection")
	} else {
		vs.state = attendStateEvent
		r.registry[m.Author.ID] = vs
	}

	event := model.Event{}
	events, err := event.GetAll(r.pool)
	if err != nil {
		return err
	}

	var eventString []string

	for i, e := range events {
		r.eventReg[m.Author.ID][i] = e
		eventString = append(eventString, fmt.Sprintf("%d. %s %s", i, e.Title, e.EventTime.Format(time.RFC822)))
	}
	err = sendMessage(s, c, fmt.Sprintf("What event are you signing up for?\n%s", strings.Join(eventString, "\n")))
	if err != nil {
		return err
	}

	return nil
}

func (r *AttendanceProvider) eventAck(c *discordgo.Channel, s *discordgo.Session, m *discordgo.MessageCreate) error {
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
		vs.state = attendStateDone
		r.registry[m.Author.ID] = vs
	}

	var (
		chosenChar  model.Character
		chosenEvent model.Event
	)

	for _, v := range r.charReg[m.Author.ID] {
		if v.Id == r.registry[m.Author.ID].characterId {
			chosenChar = v
		}

	}

	for _, v := range r.eventReg[m.Author.ID] {
		if v.Id == r.registry[m.Author.ID].eventId {
			chosenEvent = v
		}
	}

	err = sendMessage(s, c, fmt.Sprintf("Does this all look correct?\nCharacter: %s\nEvent: %s\n1. Yes\n2. No",
		chosenChar.Name,
		chosenEvent.Title,
	))

	if err != nil {
		return err
	}
	return nil

}

func (r *AttendanceProvider) doneAck(s *discordgo.Session, c *discordgo.Channel, m *discordgo.MessageCreate) error {
	if m.Content == "1" {
		dat := r.registry[m.Author.ID]
		err := dat.toModel().Save(r.pool)

		if err != nil {
			return err
		}

		if err = sendMessage(s, c, "You're all signed up."); err != nil {
			return err
		}

		r.reset(m)
		return nil
	} else if m.Content == "2" {
		if err := sendMessage(s, c, "Resetting all your information"); err != nil {
			return err
		}

		r.reset(m)
		r.init(c, s, m)
		return nil
	}
	return errors.New("invalid input")

}

func (r *AttendanceProvider) reset(m *discordgo.MessageCreate) {
	delete(r.registry, m.Author.ID)
	delete(r.charReg, m.Author.ID)
	delete(r.eventReg, m.Author.ID)
}
