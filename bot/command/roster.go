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
	"strings"
	"time"
)

const (
	rosterStateStart = 0
	rosterStatePrint = 1
	rosterStateDone  = 2
)

type RosterState struct {
	eventId int64
	state   int64
	userId  string
}

func (r *RosterState) IsComplete() bool {
	return r.state == rosterStateDone && r.eventId != 0
}

type RosterProvider struct {
	pool     *pgxpool.Pool
	registry map[string]RosterState
	manifest *Manifest
	eventReg map[string]map[int]model.Event
}

func NewRosterProvider(db *pgxpool.Pool) *RosterProvider {
	provider := &RosterProvider{
		registry: make(map[string]RosterState),
		eventReg: make(map[string]map[int]model.Event),
		pool:     db,
	}

	steps := []Step{
		provider.start,
		provider.done,
	}

	provider.manifest = &Manifest{Steps: steps}

	return provider
}

func (r *RosterProvider) Name() string {
	return "!roster"
}

func (r *RosterProvider) Description() string {
	return "returns a detailed breakdown of current event wide attendance"
}

func (r *RosterProvider) Cleanup() {
}

func (r *RosterProvider) WorkflowForUser(userId string) State {
	if v, ok := r.registry[userId]; ok {
		return &v
	} else {
		return nil
	}
}

func (r *RosterProvider) Handle(s *discordgo.Session, m *discordgo.MessageCreate) {
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
		err = sendMessage(s, c.ID, "Please restart the roster by typing **!roster**")
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

func (r *RosterProvider) start(m *discordgo.MessageCreate) (string, error) {
	if _, ok := r.registry[m.Author.ID]; !ok {
		e := model.Event{}
		events, err := e.GetAll(r.pool)
		if err != nil {
			return "", ErrorInternalError
		}

		if len(events) == 0 {
			return "", errors.New("there are no events to inspect")
		}

		r.registry[m.Author.ID] = RosterState{
			state:  rosterStatePrint,
			userId: m.Author.ID,
		}

		r.eventReg[m.Author.ID] = make(map[int]model.Event)

		var eventString []string
		for i, e := range events {
			r.eventReg[m.Author.ID][i] = e
			eventString = append(eventString, fmt.Sprintf("%d. %s %s", i, e.Title, e.EventTime.Format(time.RFC822)))
		}

		return fmt.Sprintf("What event would you like to inspect?\n%s", strings.Join(eventString, "\n")), nil
	}

	return "", nil
}

func (r *RosterProvider) done(m *discordgo.MessageCreate) (string, error) {
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
	}

	c := model.Character{}
	toons, err := c.GetAllAttendingEvent(r.pool, vs.eventId)
	if err != nil {
		return "", ErrorInternalError
	}

	statString := eq.PrintStats(eq.RaidWideClassCounts(toons))
	str := fmt.Sprintf("Summary:\n%s", statString)

	return str, nil
}
