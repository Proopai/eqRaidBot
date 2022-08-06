package bot

import (
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/bwmarrin/discordgo"
)

const (
	regStateName  = 0
	regStateClass = 1
	regStateLevel = 3
	regStateDone  = 4
	regStateSaved = 5
)

type RegistrationProvider struct {
	registry map[string]registrationState
}

func NewRegistryProvider() *RegistrationProvider {
	return &RegistrationProvider{registry: make(map[string]registrationState)}
}

type registrationState struct {
	state  int64
	Name   string
	Class  int64
	Level  int64
	userId string
}

func (r *registrationState) isComplete() bool {
	return r.state == regStateSaved && r.Name != "" && r.Class != 0 && r.Level != 0
}

func (r *RegistrationProvider) registrationStep(s *discordgo.Session, m *discordgo.MessageCreate) {
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

		err = sendMessage(s, c, fmt.Sprintf("What is your class? Respond with the number that corresponds. \n%s", classChoiceString()))
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
			_ = sendMessage(s, c, fmt.Sprintf("There was an error with your input - please try again, the current max level is %d", maxLevel))
			return
		}

		if err = sendMessage(s, c, fmt.Sprintf("Is this all correct?\nName: %s\nClass: %s\nLevel: %d\n\n1. Yes\n2. No", reg.Name, classChoiceMap[reg.Class], r.registry[m.Author.ID].Level)); err != nil {
			log.Println(err.Error())
		}
	case regStateDone:
		if m.Content == "1" {
			// save
			if err = sendMessage(s, c, "Saving your information.  You do not need to register again."); err != nil {
				log.Println(err.Error())
			}

			v := r.registry[m.Author.ID]
			v.state = regStateSaved
			r.registry[m.Author.ID] = v
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

func (r *RegistrationProvider) registrationWorkflow(userId string) *registrationState {
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

	if _, ok := classChoiceMap[classId]; !ok {
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

	if i > maxLevel {
		return errors.New("level is too high")
	}

	v := r.registry[m.Author.ID]
	v.Level = i
	v.state = regStateDone
	r.registry[m.Author.ID] = v
	return nil
}
