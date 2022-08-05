package bot

import (
	"log"
	"regexp"

	"github.com/bwmarrin/discordgo"
)

const (
	cmdRegister    = "!register"
	cmdAttend      = "!attend"
	cmdSplit       = "!split"
	cmdEvents      = "!events"
	cmdCreateEvent = "!create-event"
	cmdHelp        = "!help"
)

type Commands struct {
	registrationProvider *RegistrationProvider
	attedanceProvider    *AttendanceProvider
}

var regMatch = regexp.MustCompile("^(![a-zA-Z]+)")

func NewCommands() *Commands {
	return &Commands{
		registrationProvider: NewRegistryProvider(),
		attedanceProvider:    NewAttendanceProvider(),
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
		r.attedanceProvider.recordAttendance(s, m)
	case cmdSplit:
	case cmdEvents:
	case cmdCreateEvent:
	case cmdHelp:
		help(s, m)
	default:
		hasRegistrationState := r.registrationProvider.registrationWorkflow(m.Author.ID)
		if hasRegistrationState != nil && !hasRegistrationState.isComplete() {
			r.registrationProvider.registrationStep(s, m)
		}
	}
}

var helpMessage = `Eq Raid Bot is a discord based EverQuest raid helper. 
Please refer to the list of commands below. 

**!register** 		- prompts the bot to begin a workflow which allows a user to registers ones characters
**!attend**   		- prompts the user to confirm their attendance to an event
**!split**    		- splits registered members into N balanced groups for an event
**!events**   		- lists events
**!create-event**   - creates an event
**!help**     	    - shows this message
`

func help(s *discordgo.Session, m *discordgo.MessageCreate) {
	_, err := s.ChannelMessageSend(m.ChannelID, helpMessage)
	if err != nil {
		log.Print(err.Error())
		return
	}
}
