package command

import (
	"github.com/bwmarrin/discordgo"
	"log"
)

type commandAction int

const (
	actionError = commandAction(1)
	actionSent  = commandAction(2)
	actionSkip  = commandAction(3)

	Register      = "!register"
	MyCharacters  = "!my-characters"
	EditCharacter = "!edit-characters"
	Withdraw      = "!withdraw"
	Split         = "!split"
	ListEvents    = "!event-list"
	CreateEvent   = "!event-create"
	Roster        = "!roster"
	Help          = "!help"
)

type Provider interface {
	Name() string
	Description() string
	Handle(s *discordgo.Session, m *discordgo.MessageCreate)
	WorkflowForUser(userId string) State
	Cleanup()
}

type StateRegistry map[string]State

type State interface {
	IsComplete() bool
	Step() int64
}

type Manifest struct {
	Steps []Step
}

type Step func(m *discordgo.MessageCreate) (string, error)

func actionCommandManifest(manifest *Manifest, state int64, m *discordgo.MessageCreate) (string, error) {

	if state < 0 || state > int64(len(manifest.Steps)-1) {
		return "", ErrorInternalError
	}
	res, err := manifest.Steps[state](m)
	if err != nil {
		return "", err
	}

	return res, nil
}

func genericSimpleHandler(s *discordgo.Session, m *discordgo.MessageCreate, manifest *Manifest) {
	c, err := s.UserChannelCreate(m.Author.ID)
	if err != nil {
		log.Print(err.Error())
		return
	}

	if _, err := processCommand(manifest, 0, m, s, c.ID); err != nil {
		log.Println(err.Error())
	}
}

func genericStepwiseHandler(s *discordgo.Session, m *discordgo.MessageCreate, manifest *Manifest, registry StateRegistry) {
	c, err := s.UserChannelCreate(m.Author.ID)
	if err != nil {
		log.Print(err.Error())
		return
	}

	action, err := processCommand(manifest, 0, m, s, c.ID)
	if action == actionSent || actionError == action {
		return
	}

	if _, ok := registry[m.Author.ID]; !ok {
		err = sendMessage(s, c.ID, "Please restart the split process by typing **!split**")
		if err != nil {
			return
		}
	}

	reg := registry[m.Author.ID]

	_, err = processCommand(manifest, reg.Step(), m, s, c.ID)
	if err != nil {
		log.Println(err.Error())
	}
}

func processCommand(manifest *Manifest, state int64, m *discordgo.MessageCreate, s *discordgo.Session, cId string) (commandAction, error) {
	var (
		msg string
		err error
	)

	if msg, err = actionCommandManifest(manifest, state, m); err != nil {
		err = sendMessage(s, cId, err.Error())
		if err != nil {
			return 0, err
		}
		return actionError, nil
	} else if msg != "" {
		if len(msg) >= 2000 {
			size := 1000
			for _, m := range chunkMsg([]rune(msg), size) {
				err = sendMessage(s, cId, m)
				if err != nil {
					return 0, err
				}
			}
		} else {
			err = sendMessage(s, cId, msg)
			if err != nil {
				return 0, err
			}
		}

		return actionSent, nil
	}
	return actionSkip, nil
}

func sendMessage(s *discordgo.Session, channelId string, msg string) error {
	_, err := s.ChannelMessageSend(channelId, msg)
	if err != nil {
		log.Print(err.Error())
		return err
	}

	return nil
}

// find midpoint via size and scan forward for a new line
func chunkMsg(slice []rune, size int) []string {
	var (
		pieces     []string
		breakpoint int
	)

	for {
		if len(slice) <= size {
			pieces = append(pieces, string(slice))
			break
		}

		for idx, v := range slice[size:] {
			if v == '\n' {
				breakpoint = idx + size
				break
			}
		}

		if breakpoint >= len(slice) {
			pieces = append(pieces, string(slice))
			break
		}

		pieces = append(pieces, string(slice[0:breakpoint+1]))
		slice = slice[breakpoint+1:]
	}

	return pieces

}
