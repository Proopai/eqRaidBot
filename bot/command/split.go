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
	splitStateStart = 0
	splitStateEvent = 1
	splitStateSplit = 2
	splitStateDone  = 3
)

type SplitState struct {
	eventId int64
	state   int64
	userId  string
}

func (r *SplitState) IsComplete() bool {
	return r.state == splitStateDone && r.eventId != 0
}

type SplitProvider struct {
	pool     *pgxpool.Pool
	registry map[string]SplitState
	eventReg map[string]map[int]model.Event
	manifest *Manifest
}

func NewSplitProvider(db *pgxpool.Pool) *SplitProvider {
	provider := &SplitProvider{
		pool:     db,
		eventReg: make(map[string]map[int]model.Event),
		registry: make(map[string]SplitState),
	}

	steps := []Step{
		provider.start,
		provider.event,
		provider.split,
	}

	provider.manifest = &Manifest{Steps: steps}

	return provider
}

func (r *SplitProvider) Name() string {
	return "!split"
}

func (r *SplitProvider) Description() string {
	return "splits a raid force into N separate forces"
}

func (r *SplitProvider) Cleanup() {
}

func (r *SplitProvider) WorkflowForUser(userId string) State {
	if v, ok := r.registry[userId]; ok {
		return &v
	} else {
		return nil
	}
}

func (r *SplitProvider) Handle(s *discordgo.Session, m *discordgo.MessageCreate) {
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
		err = sendMessage(s, c.ID, "Please restart the split process by typing **!split**")
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

func (r *SplitProvider) start(m *discordgo.MessageCreate) (string, error) {
	// check the database to see if they have previously registered
	if _, ok := r.registry[m.Author.ID]; !ok {
		e := model.Event{}
		events, err := e.GetAll(r.pool)
		if err != nil {
			return "", ErrorInternalError
		}

		if len(events) == 0 {
			return "", errors.New("there are no events to split")
		}

		r.registry[m.Author.ID] = SplitState{
			state:  splitStateEvent,
			userId: m.Author.ID,
		}

		r.eventReg[m.Author.ID] = make(map[int]model.Event)

		var eventString []string
		for i, e := range events {
			r.eventReg[m.Author.ID][i] = e
			eventString = append(eventString, fmt.Sprintf("%d. %s %s", i, e.Title, e.EventTime.Format(time.RFC822)))
		}

		return fmt.Sprintf("What event would you like to split?\n%s", strings.Join(eventString, "\n")), nil
	}

	return "", nil
}

func (r *SplitProvider) event(m *discordgo.MessageCreate) (string, error) {
	i, err := strconv.Atoi(m.Content)
	if err != nil {
		return "", ErrorInvalidInput
	}

	vs := r.registry[m.Author.ID]

	for k, v := range r.eventReg[m.Author.ID] {
		if k == i {
			vs.eventId = v.Id
			break
		}
	}

	if vs.eventId == 0 {
		return "", errors.New("invalid event selection")
	} else {
		vs.state = splitStateSplit
		r.registry[m.Author.ID] = vs
	}

	return "How many ways should I split this event? e.g. 4", nil
}

func (r *SplitProvider) split(m *discordgo.MessageCreate) (string, error) {
	i, err := strconv.Atoi(m.Content)
	if err != nil {
		return "", ErrorInvalidInput
	}

	if i < 2 {
		// implement group generation
		return "", errors.New("you cannot one split an event")
	}

	a := model.Attendance{}

	attendees, err := a.GetAttendees(r.pool, r.registry[m.Author.ID].eventId)
	if err != nil {
		return "", ErrorInternalError
	}

	var splitString string

	splitter := eq.NewSplitter(attendees, false)
	splits := splitter.Split(i)

	for raidI, split := range splits {
		splitString += fmt.Sprintf("*** ===> Raid %d <===***\n", raidI+1)
		for g, group := range split {
			splitString += fmt.Sprintf("** -- Group %d -- **\n", g+1)
			var items []string
			for j, c := range group {
				if j == 0 {
					items = append(items, fmt.Sprintf("*%s - %s*", eq.ClassChoiceMap[c.Class], c.Name))
				} else {
					items = append(items, fmt.Sprintf("%s - %s", eq.ClassChoiceMap[c.Class], c.Name))
				}
			}
			splitString += strings.Join(items, ", ") + "\n"
		}
	}

	r.reset(m)

	return splitString, nil
}

func (r *SplitProvider) reset(m *discordgo.MessageCreate) {
	delete(r.registry, m.Author.ID)
}
