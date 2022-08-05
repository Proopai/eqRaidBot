package bot

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type AttendanceProvider struct {
}

func NewAttendanceProvider() *AttendanceProvider {
	return &AttendanceProvider{}
}

func (r *AttendanceProvider) recordAttendance(s *discordgo.Session, m *discordgo.MessageCreate) {
	spaceIdx := strings.Index(m.Content, " ")

	c := m.Content[spaceIdx:]

	// look up the event
	// mark this perosn as attending

	channel, err := s.UserChannelCreate(m.Author.ID)
	if err != nil {
		log.Print(err.Error())
		return
	}

	if err = sendMessage(s, channel, fmt.Sprintf("Great! See you at %s.\n", c)); err != nil {
		log.Println(err.Error())
	}
}
