package command

import (
	"eqRaidBot/bot/eq"
	"eqRaidBot/db/model"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	regStateName  = 0
	regStateClass = 1
	regStateLevel = 3
	regStateDone  = 4
	regStateSaved = 5
)

type RegistrationProvider struct {
	pool     *pgxpool.Pool
	registry map[string]registrationState
}

func NewRegistryProvider(db *pgxpool.Pool) *RegistrationProvider {
	return &RegistrationProvider{
		pool:     db,
		registry: make(map[string]registrationState),
	}
}

type registrationState struct {
	state  int64
	Name   string
	Class  int64
	Level  int64
	userId string
}

func (r *registrationState) toModel() *model.Character {
	return &model.Character{
		Name:      r.Name,
		Class:     r.Class,
		Level:     r.Level,
		AA:        0,
		IsBot:     false,
		CreatedBy: r.userId,
	}
}

func (r *registrationState) IsComplete() bool {
	return r.state == regStateSaved && r.Name != "" && r.Class != 0 && r.Level != 0
}

func (r *RegistrationProvider) MyCharacters(s *discordgo.Session, m *discordgo.MessageCreate) {
	c, err := s.UserChannelCreate(m.Author.ID)
	if err != nil {
		log.Print(err.Error())
		return
	}

	char := model.Character{}
	toons, err := char.GetByOwner(r.pool, m.Author.ID)
	if err != nil {
		log.Println(err.Error())
		_ = sendMessage(s, c, "There was a problem with finding your characters!")
		return
	}

	var charStrings []string

	for i, k := range toons {
		charStrings = append(charStrings, fmt.Sprintf("%d. %s - %d %s %s", i+1, k.Name, k.Level, eq.ClassChoiceMap[k.Class], ""))
	}
	sendMessage(s, c, strings.Join(charStrings, "\n"))
}

func (r *RegistrationProvider) Step(s *discordgo.Session, m *discordgo.MessageCreate) {
	c, err := s.UserChannelCreate(m.Author.ID)
	if err != nil {
		log.Print(err.Error())
		return
	}

	if actioned := r.init(c, s, m); actioned {
		return
	}

	if _, ok := r.registry[m.Author.ID]; !ok {
		err = sendMessage(s, c, "Please restart the registration process by typing **!register** to the bot in the main channel")
		if err != nil {
			return
		}
	}

	reg := r.registry[m.Author.ID]

	switch reg.state {
	case regStateName:
		r.nameAck(m)

		err = sendMessage(s, c, fmt.Sprintf("What is your class? Respond with the number that corresponds. \n%s", eq.ClassChoiceString()))
		if err != nil {
			log.Print(err.Error())
		}
	case regStateClass:
		err = r.classAck(m)
		if err != nil {
			_ = sendMessage(s, c, "There was an error with your input - please try again")
			return
		}

		if err = sendMessage(s, c, fmt.Sprintf("What is your level?\n")); err != nil {
			log.Println(err.Error())
		}

	case regStateLevel:
		err = r.levelAck(m)
		if err != nil {
			_ = sendMessage(s, c, fmt.Sprintf("There was an error with your input - please try again, the current max level is %d", eq.MaxLevel))
			return
		}

		if err = sendMessage(s, c, fmt.Sprintf("Is this all correct?\nName: %s\nClass: %s\nLevel: %d\n\n1. Yes\n2. No", reg.Name, eq.ClassChoiceMap[reg.Class], r.registry[m.Author.ID].Level)); err != nil {
			log.Println(err.Error())
		}
	case regStateDone:
		if m.Content == "1" {

			dat := r.registry[m.Author.ID]
			err := dat.toModel().Save(r.pool)

			if err != nil {
				log.Println(err.Error())
				_ = sendMessage(s, c, "There was an error with your input - please try again")
				return
			}

			if err = sendMessage(s, c, "Saved your information.  You do not need to register this character again."); err != nil {
				log.Println(err.Error())
			}

			r.reset(m)
		} else if m.Content == "2" {
			if err = sendMessage(s, c, "Resetting all your information"); err != nil {
				log.Println(err.Error())
			}

			r.reset(m)
			r.init(c, s, m)
		} else {
			_ = sendMessage(s, c, "There was an error with your input - please try again")
		}

	}

}

func (r *RegistrationProvider) RegistrationWorkflow(userId string) *registrationState {
	if v, ok := r.registry[userId]; ok {
		return &v
	} else {
		return nil
	}
}

func (r *RegistrationProvider) init(c *discordgo.Channel, s *discordgo.Session, m *discordgo.MessageCreate) bool {
	// check the database to see if they have previously registered
	if _, ok := r.registry[m.Author.ID]; !ok {
		r.registry[m.Author.ID] = registrationState{
			state:  regStateName,
			userId: m.Author.ID,
		}
		if err := sendMessage(s, c, fmt.Sprintf("Hello %s, what is your characters name?", m.Author.Username)); err != nil {
			return false
		}
		return true
	}

	return false
}

func (r *RegistrationProvider) reset(m *discordgo.MessageCreate) {
	delete(r.registry, m.Author.ID)
}

func (r *RegistrationProvider) nameAck(m *discordgo.MessageCreate) {
	v := r.registry[m.Author.ID]
	v.Name = m.Content
	v.state = regStateClass
	r.registry[m.Author.ID] = v
}

func (r *RegistrationProvider) classAck(m *discordgo.MessageCreate) error {
	v := r.registry[m.Author.ID]

	classId, err := strconv.ParseInt(m.Content, 10, 64)
	if err != nil {
		return err
	}

	if _, ok := eq.ClassChoiceMap[classId]; !ok {
		return errors.New("invalid class")
	}

	v.Class = classId
	v.state = regStateLevel
	r.registry[m.Author.ID] = v

	return nil
}

func (r *RegistrationProvider) levelAck(m *discordgo.MessageCreate) error {
	i, err := strconv.ParseInt(m.Content, 10, 64)
	if err != nil {
		return err
	}

	if i > eq.MaxLevel {
		return errors.New("level is too high")
	}

	v := r.registry[m.Author.ID]
	v.Level = i
	v.state = regStateDone
	r.registry[m.Author.ID] = v
	return nil
}
