package command

import (
	"eqRaidBot/bot/eq"
	"eqRaidBot/db/model"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	rosterStateStart = 0
	rosterStatePrint = 1
	rosterStateDone  = 2
)

type rosterState struct {
	eventId int64
	state   int64
	userId  string
	ttl     time.Time
}

func (r *rosterState) IsComplete() bool {
	return r.state == rosterStateDone && r.eventId != 0
}

func (r *rosterState) Step() int64 {
	return r.state
}

func (r *rosterState) TTL() time.Time {
	return r.ttl
}

type RosterProvider struct {
	pool     *pgxpool.Pool
	registry StateRegistry
	manifest *Manifest
	eventReg map[string]map[int]model.Event
}

func NewRosterProvider(db *pgxpool.Pool) *RosterProvider {
	provider := &RosterProvider{
		registry: make(StateRegistry),
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
	return Roster
}

func (r *RosterProvider) Description() string {
	return "returns a detailed breakdown of current event wide attendance"
}

func (r *RosterProvider) Cleanup() {
	cleanupCache(r.registry, func(k string) {
		delete(r.registry, k)
		delete(r.eventReg, k)
	})
}

func (r *RosterProvider) WorkflowForUser(userId string) State {
	if v, ok := r.registry[userId]; ok {
		return v
	} else {
		return nil
	}
}

func (r *RosterProvider) Handle(s *discordgo.Session, m *discordgo.MessageCreate) {
	genericStepwiseHandler(s, m, r.manifest, r.registry)
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

		r.registry[m.Author.ID] = &rosterState{
			state:  rosterStatePrint,
			userId: m.Author.ID,
			ttl:    time.Now().Add(commandCacheWindow),
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
			vs.(*rosterState).eventId = v.Id
			break
		}
	}

	if vs.(*rosterState).eventId == 0 {
		return "", errors.New("invalid event selection")
	}

	c := model.Character{}
	toons, err := c.GetAllAttendingEvent(r.pool, vs.(*rosterState).eventId)
	if err != nil {
		return "", ErrorInternalError
	}

	statString := eq.PrintStats(eq.RaidWideClassCounts(toons))

	var (
		mC, bC             int
		boxString, mString []string
	)

	for _, t := range toons {
		if t.CharacterType == model.TypeBox {
			boxString = append(boxString, fmt.Sprintf("(%s)%s", eq.ClassAbbreviationsMap[t.Class], t.Name))
			bC++
		} else {
			mString = append(mString, fmt.Sprintf("(%s)%s", eq.ClassAbbreviationsMap[t.Class], t.Name))
			mC++
		}
	}

	sort.Strings(boxString)
	sort.Strings(mString)

	str := fmt.Sprintf("__Summary__:\n%s\n**Mains** - %d: %s \n **Boxes** - %d: %s",
		statString,
		mC,
		strings.Join(mString, ", "),
		bC,
		strings.Join(boxString, ", "))

	r.Reset(m)

	return str, nil
}

func (r *RosterProvider) Reset(m *discordgo.MessageCreate) {
	delete(r.registry, m.Author.ID)
	delete(r.eventReg, m.Author.ID)
}
