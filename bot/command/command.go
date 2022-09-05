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
)

type Provider interface {
	Name() string
	Description() string
	Handle(s *discordgo.Session, m *discordgo.MessageCreate)
	WorkflowForUser(userId string) State
	Cleanup()
}

type State interface {
	IsComplete() bool
}

type Manifest struct {
	Steps []Step
}

type Step func(m *discordgo.MessageCreate) (string, error)

func actionCommandManifest(manifest *Manifest, state int64, m *discordgo.MessageCreate) (string, error) {
	res, err := manifest.Steps[state](m)
	if err != nil {
		return "", err
	}

	return res, nil
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
