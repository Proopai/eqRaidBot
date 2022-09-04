package bot

import (
	"eqRaidBot/bot/command"
	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
	"log"
	"regexp"
)

const (
	cmdRegister        = "!register"
	cmdMyCharacters    = "!my-characters"
	cmdRemoveCharacter = "!remove-character"
	cmdWithdraw        = "!withdraw"
	cmdSplit           = "!split"
	cmdListEvents      = "!list-events"
	cmdCreateEvent     = "!create-event"
	cmdRoster          = "!roster"
	cmdHelp            = "!help"
)

type CommandController struct {
	providers map[string]command.Provider
}

var regMatch = regexp.MustCompile("^(![a-zA-Z]+-?[a-zA-Z]+)")

func NewCommandController(db *pgxpool.Pool) *CommandController {
	providerMap := make(map[string]command.Provider)
	providers := []command.Provider{
		command.NewMyCharactersProvider(db),
		command.NewRegistrationProvider(db),
		command.NewListEventsProvider(db),
		command.NewCreateEventProvider(db),
	}

	for _, p := range providers {
		go p.Cleanup()
	}

	for _, p := range providers {
		providerMap[p.Name()] = p
	}

	return &CommandController{
		providers: providerMap,
	}
}

func (r *CommandController) MessageCreatedHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	cmd := regMatch.FindString(m.Content)

	// only switch on valid commands
	switch cmd {
	case cmdRegister, cmdMyCharacters, cmdListEvents, cmdCreateEvent:
		r.providers[cmd].Handle(s, m)
	case cmdHelp:
		help(s, m)
	default:
		for _, p := range r.providers {
			state := p.WorkflowForUser(m.Author.ID)
			if state == nil {
				continue
			}

			if !state.IsComplete() {
				p.Handle(s, m)
				return
			}
		}
	}
}

var helpMessage = `Eq Raid Bot is a discord based EverQuest raid helper. 
Please refer to the list of commands below. 

**!register** 		  - prompts the bot to begin a workflow which allows a user to registers ones characters.
**!my-characters** 	  - shows the users registered characters.
**!remove-character** - deletes a character from the list of selectable characters for a given user. (wip)
**!withdraw**   	  - allows the user to reneg on a event they signed up for. (wip)
**!split**    		  - splits registered members into N balanced groups for an event
**!list-events**      - lists events.
**!create-event**     - prompts the bot to begin the create event workflow
**!help**     	      - shows this message
`

func help(s *discordgo.Session, m *discordgo.MessageCreate) {
	_, err := s.ChannelMessageSend(m.ChannelID, helpMessage)
	if err != nil {
		log.Print(err.Error())
		return
	}
}
