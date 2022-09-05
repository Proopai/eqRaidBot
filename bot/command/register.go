package command

import (
	"eqRaidBot/bot/eq"
	"eqRaidBot/db/model"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
	"log"
	"strconv"
)

const (
	regStateStart = 0
	regStateName  = 1
	regStateClass = 2
	regStateLevel = 3
	regStateMata  = 4
	regStateDone  = 5

	typeBox  = 1
	typeMain = 2
	typeAlt  = 3
)

var charTypeMap = map[int64]string{
	typeBox:  "Box",
	typeMain: "Main",
	typeAlt:  "Alt",
}

type registrationState struct {
	state    int64
	name     string
	class    int64
	level    int64
	userId   string
	charType int64
}

func (r *registrationState) toModel() *model.Character {
	return &model.Character{
		Name:          r.name,
		Class:         r.class,
		Level:         r.level,
		AA:            0,
		CharacterType: r.charType,
		CreatedBy:     r.userId,
	}
}

func (r *registrationState) IsComplete() bool {
	return r.name != "" && r.class != 0 && r.level != 0
}

type RegistrationProvider struct {
	pool     *pgxpool.Pool
	registry map[string]registrationState
	manifest *Manifest
}

func NewRegistrationProvider(db *pgxpool.Pool) *RegistrationProvider {
	provider := &RegistrationProvider{
		pool:     db,
		registry: make(map[string]registrationState),
	}

	steps := []Step{
		provider.start,
		provider.name,
		provider.class,
		provider.level,
		provider.meta,
		provider.done,
	}

	provider.manifest = &Manifest{Steps: steps}

	return provider
}

func (r *RegistrationProvider) Name() string {
	return "!register"
}

func (r *RegistrationProvider) Description() string {
	return "begins a workflow that allows the user to register their characters"
}

func (r *RegistrationProvider) Cleanup() {
}

func (r *RegistrationProvider) Handle(s *discordgo.Session, m *discordgo.MessageCreate) {
	c, err := s.UserChannelCreate(m.Author.ID)
	if err != nil {
		log.Print(err.Error())
		return
	}

	action, _ := processCommand(r.manifest, 0, m, s, c.ID)
	if action == actionSent {
		return
	}

	if _, ok := r.registry[m.Author.ID]; !ok {
		err = sendMessage(s, c.ID, "Please restart the registration process by typing **!register**")
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

func (r *RegistrationProvider) WorkflowForUser(userId string) State {
	if v, ok := r.registry[userId]; ok {
		return &v
	} else {
		return nil
	}
}

func (r *RegistrationProvider) start(m *discordgo.MessageCreate) (string, error) {
	// check the database to see if they have previously registered
	if _, ok := r.registry[m.Author.ID]; !ok {
		r.registry[m.Author.ID] = registrationState{
			state:  regStateName,
			userId: m.Author.ID,
		}

		return fmt.Sprintf("Hello %s, what is your characters name?", m.Author.Username), nil
	}

	return "", nil
}

func (r *RegistrationProvider) name(m *discordgo.MessageCreate) (string, error) {
	v := r.registry[m.Author.ID]
	v.name = m.Content
	v.state = regStateClass
	r.registry[m.Author.ID] = v

	return fmt.Sprintf("What is your class? Respond with the number that corresponds. \n%s", eq.ClassChoiceString()), nil
}

func (r *RegistrationProvider) class(m *discordgo.MessageCreate) (string, error) {
	classId, err := strconv.ParseInt(m.Content, 10, 64)
	if err != nil {
		log.Println(err.Error())
		return "", ErrorInvalidInput
	}

	if _, ok := eq.ClassChoiceMap[classId]; !ok {
		return "", errors.New("invalid class choice, please try again and pick the number next the corresponding class")
	}

	v := r.registry[m.Author.ID]
	v.class = classId
	v.state = regStateLevel
	r.registry[m.Author.ID] = v

	return fmt.Sprintf("What is your level?\n"), nil
}

func (r *RegistrationProvider) level(m *discordgo.MessageCreate) (string, error) {
	i, err := strconv.ParseInt(m.Content, 10, 64)
	if err != nil {
		return "", ErrorInvalidInput
	}

	if i > eq.MaxLevel || i < 0 {
		return "", errors.New(fmt.Sprintf("a characters level must be between 0 and %d", eq.MaxLevel))
	}

	v := r.registry[m.Author.ID]
	v.level = i
	v.state = regStateMata
	r.registry[m.Author.ID] = v

	return "How would you describe this character?\n1. Box\n2. Main\n3. Alt", nil
}

func (r *RegistrationProvider) meta(m *discordgo.MessageCreate) (string, error) {
	if m.Content != "1" && m.Content != "2" && m.Content != "3" {
		return "", errors.New("there was a problem with your input - valid choices are 1, 2 or 3")
	}

	typeId, err := strconv.ParseInt(m.Content, 10, 64)
	if err != nil {
		return "", ErrorInvalidInput
	}

	v := r.registry[m.Author.ID]
	v.charType = typeId
	v.state = regStateDone
	r.registry[m.Author.ID] = v

	return fmt.Sprintf("Is this all correct?\nName: %s\nClass: %s\nLevel: %d\nType:%s\n\n1. Yes\n2. No",
		v.name,
		eq.ClassChoiceMap[v.class],
		v.level,
		charTypeMap[v.charType]), nil
}

func (r *RegistrationProvider) done(m *discordgo.MessageCreate) (string, error) {
	switch m.Content {
	case "1":
		dat := r.registry[m.Author.ID]
		err := dat.toModel().Save(r.pool)

		if err != nil {
			return "", ErrorInternalError
		}

		r.reset(m)

		return "Saved your information.  You do not need to register this character again.", nil
	case "2":
		r.reset(m)
		return "Resetting all your information", nil
	default:
		return "", ErrorInvalidInput
	}
}

func (r *RegistrationProvider) reset(m *discordgo.MessageCreate) {
	delete(r.registry, m.Author.ID)
}
