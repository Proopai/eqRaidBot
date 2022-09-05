package bot

import (
	"eqRaidBot/bot/command"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
	"log"
	"regexp"
	"sort"
	"strings"
)

const (
	cmdRegister        = "!register"
	cmdMyCharacters    = "!my-characters"
	cmdRemoveCharacter = "!remove-character"
	cmdWithdraw        = "!withdraw"
	cmdSplit           = "!split"
	cmdListEvents      = "!event-list"
	cmdCreateEvent     = "!event-create"
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
		command.NewSplitProvider(db),
		command.NewRosterProvider(db),
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
	case cmdRegister, cmdMyCharacters, cmdListEvents, cmdCreateEvent, cmdSplit, cmdRoster:
		r.providers[cmd].Handle(s, m)
	case cmdHelp:
		r.help(s, m)
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

var helpMessage = `Eq Raid Bot is a discord based EverQuest raid helper. Its primary goal is to track and generate raid splits.
__Please refer to the list of commands below.__ 
--------------------------------------------------------------
%s
`

func (r *CommandController) help(s *discordgo.Session, m *discordgo.MessageCreate) {
	var (
		names   []string
		longest int
	)
	for k := range r.providers {
		if len(k) > longest {
			longest = len(k)
		}
		names = append(names, k)
	}

	sort.Strings(names)

	cmdListString := ""

	for _, name := range names {
		p := r.providers[name]
		padding := strings.Repeat(" ", longest-len(name))
		subStr := fmt.Sprintf("**%s**%s  -  %s\n", p.Name(), padding, p.Description())
		cmdListString = fmt.Sprintf("%s%s", cmdListString, subStr)
	}

	_, err := s.ChannelMessageSend(m.ChannelID, fmt.Sprintf(helpMessage, cmdListString))
	if err != nil {
		log.Print(err.Error())
		return
	}
}
