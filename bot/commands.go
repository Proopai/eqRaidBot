package bot

import (
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
	registrationProvider *RegistrationProvider
	attedanceProvider    *AttendanceProvider
	eventProvider        *EventProvider
}

var regMatch = regexp.MustCompile("^(![a-zA-Z]+-?[a-zA-Z]+)")

func NewCommands(db *pgxpool.Pool) *Commands {
	return &Commands{
		registrationProvider: NewRegistryProvider(db),
		attedanceProvider:    NewAttendanceProvider(),
		eventProvider:        NewEventProvider(db),
	}
}

func (r *Commands) MessageCreated(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	cmd := regMatch.FindString(m.Content)

	switch cmd {
	case cmdRegister:
		r.registrationProvider.registrationStep(s, m)
	case cmdAttend:
		r.attedanceProvider.attendanceStep(s, m)
	case cmdSplit:
	case cmdListEvents:
		r.eventProvider.listEvents(s, m)
	case cmdCreateEvent:
		r.eventProvider.createEventStep(s, m)
	case cmdHelp:
		help(s, m)
	default:
		inProgressRegistration := r.registrationProvider.registrationWorkflow(m.Author.ID)
		if inProgressRegistration != nil && !inProgressRegistration.isComplete() {
			r.registrationProvider.registrationStep(s, m)
			return
		}

		inProgressEventCreate := r.eventProvider.eventWorkflow(m.Author.ID)
		if inProgressEventCreate != nil && !inProgressEventCreate.isComplete() {
			r.eventProvider.createEventStep(s, m)
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
