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

type splitState struct {
	eventId int64
	state   int64
	userId  string
	ttl     time.Time
}

func (r *splitState) IsComplete() bool {
	return r.state == splitStateDone && r.eventId != 0
}

func (r *splitState) Step() int64 {
	return r.state
}

func (r *splitState) TTL() time.Time {
	return r.ttl
}

type SplitProvider struct {
	pool     *pgxpool.Pool
	registry StateRegistry
	eventReg map[string]map[int]model.Event
	manifest *Manifest
}

func NewSplitProvider(db *pgxpool.Pool) *SplitProvider {
	provider := &SplitProvider{
		pool:     db,
		eventReg: make(map[string]map[int]model.Event),
		registry: make(StateRegistry),
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
	return Split
}

func (r *SplitProvider) Description() string {
	return "splits a raid force into N separate forces, not available to all users"
}

func (r *SplitProvider) Cleanup() {
	cleanupCache(r.registry, func(k string) {
		delete(r.registry, k)
		delete(r.eventReg, k)
	})
}

func (r *SplitProvider) WorkflowForUser(userId string) State {
	if v, ok := r.registry[userId]; ok {
		return v
	} else {
		return nil
	}
}

func (r *SplitProvider) Handle(s *discordgo.Session, m *discordgo.MessageCreate) {
	if !isAllowed(m) {
		err := sendMessage(s, m.ChannelID, "Only authorized users are allowed to generate splits.")
		if err != nil {
			log.Print(err.Error())
		}
		return
	}
	genericStepwiseHandler(s, m, r.manifest, r.registry)
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

		r.registry[m.Author.ID] = &splitState{
			state:  splitStateEvent,
			userId: m.Author.ID,
			ttl:    time.Now().Add(commandCacheWindow),
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
			vs.(*splitState).eventId = v.Id
			break
		}
	}

	if vs.(*splitState).eventId == 0 {
		return "", errors.New("invalid event selection")
	} else {
		vs.(*splitState).state = splitStateSplit
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

	attendees, err := a.GetAttendees(r.pool, r.registry[m.Author.ID].(*splitState).eventId)
	if err != nil {
		return "", ErrorInternalError
	}

	if len(attendees) == 0 {
		r.Reset(m)
		return "No one is coming to this event.  Try agian when more people have registered.", nil
	}

	var splitString string

	splitter := eq.NewSplitter(attendees, false)
	splits, stats := splitter.Split(i)

	for raidI, split := range splits {
		splitString += fmt.Sprintf("\n*** ===> Raid %d <===***\n", raidI+1)
		splitString += fmt.Sprintf("%s\n", eq.PrintStats(stats[raidI]))
		for g, group := range split {
			splitString += fmt.Sprintf("__Group %d__\n", g+1)
			var items []string
			for _, c := range group {
				if c.CharacterType == model.TypeBox {
					items = append(items, fmt.Sprintf("%s(box)", c.Name))
				} else {
					items = append(items, c.Name)
				}
			}
			splitString += strings.Join(items, ", ") + "\n"
		}
	}

	r.Reset(m)

	return splitString, nil
}

func (r *SplitProvider) Reset(m *discordgo.MessageCreate) {
	delete(r.registry, m.Author.ID)
	delete(r.eventReg, m.Author.ID)
}
