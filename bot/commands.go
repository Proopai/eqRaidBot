package bot

import (
	"github.com/bwmarrin/discordgo"
	"log"
)

const (
	cmdRegister = "!register"
	cmdAttend   = "!attend"
	cmdSplit    = "!split"
	cmdEvents   = "!events"
	cmdHelp     = "!help"
)

type Commands struct {
	registrationProvider *RegistrationProvider
}

func NewCommands() *Commands {
	return &Commands{registrationProvider: NewRegistryProvider()}
}

func (r *Commands) MessageCreated(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	switch m.Content {
	case cmdRegister:
		r.registrationProvider.registrationStep(s, m)
	case cmdAttend:
	case cmdSplit:
	case cmdEvents:
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

**!register** - prompts the bot to begin a workflow which allows a user to registers ones characters
**!attend**   - prompts the user to confirm their attendance to an event
**!split**    - splits registered members into N balanced groups for an event
**!events**   - lists events
**!help**     - shows this message
`

func help(s *discordgo.Session, m *discordgo.MessageCreate) {
	_, err := s.ChannelMessageSend(m.ChannelID, helpMessage)
	if err != nil {
		log.Print(err.Error())
		return
	}
}
