package command

import (
	"eqRaidBot/bot/eq"
	"eqRaidBot/db/model"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
	"log"
	"strings"
)

type WithdrawProvider struct {
	manifest *Manifest
	db       *pgxpool.Pool
}

func NewWithdrawProvider(db *pgxpool.Pool) *WithdrawProvider {
	provider := &WithdrawProvider{
		manifest: nil,
		db:       db,
	}

	steps := []Step{
		provider.start,
		provider.eventOrNext,
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
	return
}

func (p *WithdrawProvider) Handle(s *discordgo.Session, m *discordgo.MessageCreate) {
	c, err := s.UserChannelCreate(m.Author.ID)
	if err != nil {
		log.Print(err.Error())
		return
	}

	if _, err := processCommand(p.manifest, 0, m, s, c.ID); err != nil {
		log.Println(err.Error())
	}
}

func (p *WithdrawProvider) WorkflowForUser(userId string) State {
	return nil
}

func (p *WithdrawProvider) start(m *discordgo.MessageCreate) (string, error) {
	char := model.Character{}
	toons, err := char.GetByOwner(p.db, m.Author.ID)
	if err != nil {
		log.Println(err.Error())
		return "", ErrorInternalError
	}

	var charStrings []string

	for i, k := range toons {
		charStrings = append(charStrings, fmt.Sprintf("%d. %s - %d %s %s", i+1, k.Name, k.Level, eq.ClassChoiceMap[k.Class], charTypeMap[k.CharacterType]))
	}

	if len(charStrings) == 0 {
		charStrings = []string{"No characters found."}
	}

	return strings.Join(charStrings, "\n"), nil
}

func (p *WithdrawProvider) eventOrNext(m *discordgo.MessageCreate) (string, error) {
	return "", nil
}
