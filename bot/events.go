package bot

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

type EventProvider struct {
}

func NewEventProvider() *EventProvider {
	return &EventProvider{}
}

var eventListText = `All scheduled events are listed below.
1.
2.
3.
4.`

func (r *EventProvider) listEvents(s *discordgo.Session, m *discordgo.MessageCreate) {
	_, err := s.ChannelMessageSend(m.ChannelID, eventListText)
	if err != nil {
		log.Println(err.Error())
	}
}

func (r *EventProvider) createEvent(s *discordgo.Session, m *discordgo.MessageCreate) {

}
