package bot

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
	attendStateChar  = 0
	attendStateEvent = 1
	attendStateDone  = 3
)

type AttendanceProvider struct {
	pool     *pgxpool.Pool
	registry map[string]AttendanceState
	charReg  map[string]map[int]int64
	eventReg map[string]map[int]int64
	state    AttendanceState
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

func NewAttendanceProvider() *AttendanceProvider {
	return &AttendanceProvider{
		registry: make(map[string]AttendanceState),
		charReg:  make(map[string]map[int]int64),
		eventReg: make(map[string]map[int]int64),
	}
}

func (r *AttendanceProvider) attendanceStep(s *discordgo.Session, m *discordgo.MessageCreate) {
	c, err := s.UserChannelCreate(m.Author.ID)
	if err != nil {
		log.Print(err.Error())
		return
	}

	if actioned := r.init(c, s, m); actioned {
		return
	}

	att := r.registry[m.Author.ID]

	switch att.state {
	case attendStateChar:
		err := r.charAck(c, s, m)
		if err != nil {
			_ = sendMessage(s, c, "There was a problem fetching the events")
		}
	case attendStateEvent:
		err := r.eventAck(m)
		if err != nil {
			_ = sendMessage(s, c, "There was a problem fetching the events")
		}
	case attendStateDone:
		if err := r.doneAck(s, c, m); err != nil {
			_ = sendMessage(s, c, "There was an error with your input - please try again")
		}
	}

}

func (r *AttendanceProvider) init(c *discordgo.Channel, s *discordgo.Session, m *discordgo.MessageCreate) bool {
	// check the database to see if they have previously registered
	if _, ok := r.registry[m.Author.ID]; !ok {
		char := model.Character{}
		toons, err := char.GetByOwner(r.pool, m.Author.ID)
		if err != nil {
			log.Println(err.Error())
			_ = sendMessage(s, c, "There was an error with your input - please try again")
		}

		if len(toons) == 0 {
			if err := sendMessage(s, c, fmt.Sprintf("You have no characters register, please type **!register** to add one.")); err != nil {
				return false
			}
		}

		r.registry[m.Author.ID] = AttendanceState{
			state:  attendStateChar,
			userId: m.Author.ID,
		}

		r.charReg[m.Author.ID] = make(map[int]int64)
		r.eventReg[m.Author.ID] = make(map[int]int64)

		var charString []string
		for i, t := range toons {
			r.charReg[m.Author.ID][i] = t.Id
			charString = append(charString, fmt.Sprintf("%d. %s", i, t.Name))
		}

		if err := sendMessage(s, c, fmt.Sprintf("Hello %s, which character will you be brining?\n%s", m.Author.Username, strings.Join(charString, "\n"))); err != nil {
			return false
		}

		return true
	}

	return false
}

func (r *AttendanceProvider) charAck(c *discordgo.Channel, s *discordgo.Session, m *discordgo.MessageCreate) error {
	i, err := strconv.Atoi(m.Content)
	if err != nil {
		return err
	}

	for k, v := range r.charReg[m.Author.ID] {
		if k == i {
			r.state.characterId = v
			break
		}
	}

	if r.state.characterId == 0 {
		return errors.New("invalid character selection")
	} else {
		r.state.state = attendStateEvent
	}

	event := model.Event{}
	events, err := event.GetAll(r.pool)
	if err != nil {
		return err
	}

	var eventString []string

	for i, e := range events {
		r.eventReg[m.Author.ID][i] = e.Id
		eventString = append(eventString, fmt.Sprintf("%d. %s %s", i, e.Title, e.EventTime.Format(time.RFC822)))
	}
	err = sendMessage(s, c, fmt.Sprintf("What event are you signing up for?\n%s", strings.Join(eventString, "\n")))
	if err != nil {
		return err
	}

	return nil
}

func (r *AttendanceProvider) eventAck(m *discordgo.MessageCreate) error {
	i, err := strconv.Atoi(m.Content)
	if err != nil {
		return err
	}

	for k, v := range r.eventReg[m.Author.ID] {
		if k == i {
			r.state.eventId = v
			break
		}
	}

	if r.state.eventId == 0 {
		return errors.New("invalid event selection")
	} else {
		r.state.state = attendStateDone
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
	} else if m.Content == "2" {
		if err := sendMessage(s, c, "Resetting all your information"); err != nil {
			return err
		}

		r.reset(m)
		r.init(c, s, m)
	}
	return errors.New("invalid input")

}

func (r *AttendanceProvider) reset(m *discordgo.MessageCreate) {
	delete(r.registry, m.Author.ID)
	delete(r.charReg, m.Author.ID)
	delete(r.eventReg, m.Author.ID)
}
