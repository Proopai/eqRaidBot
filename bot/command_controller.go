package bot

import (
	"eqRaidBot/bot/command"
	"log"
	"regexp"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
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
	registrationProvider *command.RegistrationProvider
	attedanceProvider    *command.AttendanceProvider
	eventProvider        *command.EventProvider
	splitProvider        *command.SplitProvider

	autoAttender *AutoAttender
}

var regMatch = regexp.MustCompile("^(![a-zA-Z]+-?[a-zA-Z]+)")

func NewCommandController(db *pgxpool.Pool) *CommandController {
	return &CommandController{
		registrationProvider: command.NewRegistryProvider(db),
		attedanceProvider:    command.NewAttendanceProvider(db),
		eventProvider:        command.NewEventProvider(db),
		splitProvider:        command.NewSplitProvider(db),
		autoAttender:         NewAutoAttender(db),
	}
}

func (r *CommandController) Run(duration time.Duration) {
	ch := make(chan struct{})
	go r.autoAttender.Run(ch, duration)

}

func (r *CommandController) MessageCreatedHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	cmd := regMatch.FindString(m.Content)

	switch cmd {
	case cmdRegister:
		r.registrationProvider.Step(s, m)
	case cmdMyCharacters:
		r.registrationProvider.MyCharacters(s, m)
	case cmdRemoveCharacter:
		// @TODO
		break
	case cmdSplit:
		r.splitProvider.Step(s, m)
	case cmdListEvents:
		r.eventProvider.ListEvents(s, m)
	case cmdCreateEvent:
		r.eventProvider.CreateEventStep(s, m)
	case cmdHelp:
		help(s, m)
	default:
		// handle pick workflow
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

**!register** 		  - prompts the bot to begin a workflow which allows a user to registers ones characters.
**!my-characters** 	  - shows the users registered characters.
**!remove-character** - deletes a character from the list of selectable characters for a given user. (wip)
**!withdraw**   	  - allows the user to reneg on a event they signed up for. (wip)
**!split**    		  - splits registered members into N balanced groups for an event (wip)
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
