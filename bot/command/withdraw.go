package command

import (
	"eqRaidBot/db/model"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
	"log"
	"strings"
	"time"
)

const (
	withdrawStateStart   = 0
	withdrawStateConfirm = 1
	withdrawStateEvent   = 2
	withdrawStateDone    = 3
)

type WithdrawProvider struct {
	manifest *Manifest
	registry StateRegistry

	attReg map[string][]model.Attendance
	db     *pgxpool.Pool
}

type withdrawState struct {
	eventId int64
	state   int64
	userId  string
	ttl     time.Time
}

func (r *withdrawState) IsComplete() bool {
	return r.state == withdrawStateDone && r.eventId != 0
}

func (r *withdrawState) Step() int64 {
	return r.state
}

func (r *withdrawState) TTL() time.Time {
	return r.ttl
}

func NewWithdrawProvider(db *pgxpool.Pool) *WithdrawProvider {
	provider := &WithdrawProvider{
		registry: make(StateRegistry),
		attReg:   make(map[string][]model.Attendance),
		db:       db,
	}

	steps := []Step{
		provider.start,
		provider.eventOrNext,
		provider.handleEvent,
	}

	provider.manifest = &Manifest{Steps: steps}

	return provider
}

func (p *WithdrawProvider) Name() string {
	return Withdraw
}

func (p *WithdrawProvider) Description() string {
	return "allows the user to opt out of an event"
}

// we have no state to clean up with this command
func (p *WithdrawProvider) Cleanup() {
	cleanupCache(p.registry, func(k string) {
		delete(p.registry, k)
		delete(p.attReg, k)
	})
}

func (p *WithdrawProvider) Handle(s *discordgo.Session, m *discordgo.MessageCreate) {
	genericStepwiseHandler(s, m, p.manifest, p.registry)
}

func (p *WithdrawProvider) WorkflowForUser(userId string) State {
	if v, ok := p.registry[userId]; ok {
		return v
	} else {
		return nil
	}
}

func (p *WithdrawProvider) start(m *discordgo.MessageCreate) (string, error) {
	if _, ok := p.registry[m.Author.ID]; !ok {
		p.registry[m.Author.ID] = &withdrawState{
			state:  withdrawStateConfirm,
			userId: m.Author.ID,
			ttl:    time.Now().Add(commandCacheWindow),
		}

		//return "Please pick an option:\n1. Withdraw from the next event on all characters\n2. Withdraw from a specific event", nil
		return "Confirm your choice:\n1. Withdraw from the next event on all characters\n2. Cancel", nil
	}

	return "", nil
}

func (p *WithdrawProvider) eventOrNext(m *discordgo.MessageCreate) (string, error) {
	switch m.Content {
	case "1":
		// next event
		return p.next(m)
	case "2":
		p.Reset(m)
		return "Canceled withdraw process.", nil
	//case "3":
	//	return p.event(m)
	default:
		return "", ErrorInvalidInput
	}
}

func (p *WithdrawProvider) event(m *discordgo.MessageCreate) (string, error) {
	a := model.Attendance{}
	att, err := a.GetPendingAttendance(p.db, m.Author.ID)
	if err != nil {
		log.Println(err.Error())
		return "", ErrorInternalError
	}

	if len(att) == 0 {
		log.Println("no pending invitations")
		p.Reset(m)
		return "", errors.New("you have no pending invitations")
	}

	eMap, cMap, err := p.metaData(att)
	if err != nil {
		return "", ErrorInternalError
	}

	v := p.registry[m.Author.ID].(*withdrawState)
	v.state = withdrawStateEvent

	p.registry[m.Author.ID] = v
	p.attReg[m.Author.ID] = att

	return p.individualString(att, cMap, eMap)
}

func (p *WithdrawProvider) next(m *discordgo.MessageCreate) (string, error) {
	// find the next event and remove all characters that are registered to me from it.
	events := model.Event{}
	attendance := model.Attendance{}

	nextEvent, err := events.GetNext(p.db)
	if err != nil {
		return "", ErrorInternalError
	}

	att, err := attendance.GetMyAttendanceForEvent(p.db, nextEvent.Id, m.Author.ID)
	if err != nil {
		return "", ErrorInternalError
	}

	for i := range att {
		if !att[i].Withdrawn {
			att[i].Withdrawn = true
			err := att[i].Update(p.db)
			if err != nil {
				return "", ErrorInternalError
			}
		}
	}
	p.Reset(m)

	return fmt.Sprintf("%d attendees have been marked as absent from %s on %v", len(att), nextEvent.Title, nextEvent.EventTime), nil
}

func (p *WithdrawProvider) handleEvent(m *discordgo.MessageCreate) (string, error) {
	return "", nil
}

func (p *WithdrawProvider) individualString(att []model.Attendance, cMap map[int64]model.Character, eMap map[int64]model.Event) (string, error) {
	var msgs []string
	for idx, at := range att {
		var (
			event model.Event
			char  model.Character
		)

		if v, ok := cMap[at.CharacterId]; ok {
			char = v
		}

		if v, ok := eMap[at.EventId]; ok {
			event = v
		}

		if char.Id == 0 || event.Id == 0 {
			return "", ErrorInternalError
		}

		msgs = append(msgs, fmt.Sprintf("%d. %s as %s @ %s %s",
			idx,
			char.Name,
			model.CharTypeMap[char.CharacterType],
			event.EventTime.Format(time.RFC822),
			event.Title,
		))

	}

	return strings.Join(msgs, "\n"), nil

}

func (p *WithdrawProvider) metaData(att []model.Attendance) (map[int64]model.Event, map[int64]model.Character, error) {

	var (
		charIds  []int64
		eventIds []int64
	)

	e := model.Event{}
	c := model.Character{}

	for _, v := range att {
		charIds = append(charIds, v.CharacterId)
		eventIds = append(eventIds, v.EventId)
	}

	events, err := e.GetWhereIn(p.db, eventIds)
	if err != nil {
		log.Print(err.Error())
		return nil, nil, ErrorInternalError
	}

	characters, err := c.GetWhereIn(p.db, charIds)
	if err != nil {
		log.Print(err.Error())
		return nil, nil, ErrorInternalError
	}

	eventMap := make(map[int64]model.Event)
	charMap := make(map[int64]model.Character)

	for _, ev := range events {
		eventMap[ev.Id] = ev
	}

	for _, char := range characters {
		charMap[char.Id] = char
	}

	return eventMap, charMap, nil
}

func (p *WithdrawProvider) Reset(m *discordgo.MessageCreate) {
	delete(p.registry, m.Author.ID)
	delete(p.attReg, m.Author.ID)
}
