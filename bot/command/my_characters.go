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

type MyCharactersProvider struct {
	manifest *Manifest
	db       *pgxpool.Pool
}

func NewMyCharactersProvider(db *pgxpool.Pool) *MyCharactersProvider {
	provider := &MyCharactersProvider{
		manifest: nil,
		db:       db,
	}

	steps := []Step{
		provider.list,
	}

	provider.manifest = &Manifest{Steps: steps}

	return provider
}

func (p *MyCharactersProvider) Name() string {
	return MyCharacters
}

func (p *MyCharactersProvider) Description() string {
	return "lists all the currently registered characters for a user"
}

// we have no state to clean up with this command
func (p *MyCharactersProvider) Cleanup() {
	return
}

func (r *MyCharactersProvider) Reset(m *discordgo.MessageCreate) {
}

func (p *MyCharactersProvider) Handle(s *discordgo.Session, m *discordgo.MessageCreate) {
	genericSimpleHandler(s, m, p.manifest)
}

func (p *MyCharactersProvider) WorkflowForUser(userId string) State {
	return nil
}

func (p *MyCharactersProvider) list(m *discordgo.MessageCreate) (string, error) {
	char := model.Character{}
	toons, err := char.GetByOwner(p.db, m.Author.ID)
	if err != nil {
		log.Println(err.Error())
		return "", ErrorInternalError
	}

	var charStrings []string

	for i, k := range toons {
		charStrings = append(charStrings, fmt.Sprintf("%d. %s - %d %s %s", i+1, k.Name, k.Level, eq.ClassChoiceMap[k.Class], model.CharTypeMap[k.CharacterType]))
	}

	if len(charStrings) == 0 {
		charStrings = []string{"No characters found."}
	}

	return strings.Join(charStrings, "\n"), nil
}
