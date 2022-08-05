package bot

import (
	"github.com/bwmarrin/discordgo"
	"log"
)

func sendMessage(s *discordgo.Session, c *discordgo.Channel, msg string) error {
	_, err := s.ChannelMessageSend(c.ID, msg)
	if err != nil {
		log.Print(err.Error())
		return err
	}

	return nil
}
