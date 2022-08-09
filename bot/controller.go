package bot

import (
	"eqRaidBot/bot/command"
	"log"
	"regexp"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	cmdRegister    = "!register"
	cmdAttend      = "!attend"
	cmdSplit       = "!split"
	cmdListEvents  = "!list-events"
	cmdCreateEvent = "!create-event"
	cmdRoster      = "!roster"
	cmdHelp        = "!help"
)

type Commands struct {
	registrationProvider *command.RegistrationProvider
	attedanceProvider    *command.AttendanceProvider
	eventProvider        *command.EventProvider
	splitProvider        *command.SplitProvider
}

var regMatch = regexp.MustCompile("^(![a-zA-Z]+-?[a-zA-Z]+)")

func NewCommands(db *pgxpool.Pool) *Commands {
	return &Commands{
		registrationProvider: command.NewRegistryProvider(db),
		attedanceProvider:    command.NewAttendanceProvider(db),
		eventProvider:        command.NewEventProvider(db),
		splitProvider:        command.NewSplitProvider(db),
	}
}

func (r *Commands) MessageCreated(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	cmd := regMatch.FindString(m.Content)

	switch cmd {
	case cmdRegister:
		r.registrationProvider.Step(s, m)
	case cmdAttend:
		r.attedanceProvider.Step(s, m)
	case cmdSplit:
		r.splitProvider.Step(s, m)
	case cmdListEvents:
		r.eventProvider.ListEvents(s, m)
	case cmdCreateEvent:
		r.eventProvider.CreateEventStep(s, m)
	case cmdHelp:
		help(s, m)
	default:
		inProgressRegistration := r.registrationProvider.RegistrationWorkflow(m.Author.ID)
		if inProgressRegistration != nil && !inProgressRegistration.IsComplete() {
			r.registrationProvider.Step(s, m)
			return
		}

		inProgressEventCreate := r.eventProvider.EventWorkflow(m.Author.ID)
		if inProgressEventCreate != nil && !inProgressEventCreate.IsComplete() {
			r.eventProvider.CreateEventStep(s, m)
			return
		}

		inProgressAttend := r.attedanceProvider.AttendWorkflow(m.Author.ID)
		if inProgressAttend != nil && !inProgressAttend.IsComplete() {
			r.attedanceProvider.Step(s, m)
			return
		}

		inProgressSplit := r.splitProvider.SplitWorkflow(m.Author.ID)
		if inProgressSplit != nil && !inProgressSplit.IsComplete() {
			r.splitProvider.Step(s, m)
			return
		}
	}
}

var helpMessage = `Eq Raid Bot is a discord based EverQuest raid helper. 
Please refer to the list of commands below. 

**!register** 		- prompts the bot to begin a workflow which allows a user to registers ones characters
**!attend**   		- prompts the user to confirm their attendance to an event
**!split**    		- splits registered members into N balanced groups for an event
**!list-events**    - lists events
**!create-event**   - prompts the bot to begin the create event workflow
**!help**     	    - shows this message
`

func help(s *discordgo.Session, m *discordgo.MessageCreate) {
	_, err := s.ChannelMessageSend(m.ChannelID, helpMessage)
	if err != nil {
		log.Print(err.Error())
		return
	}
}
