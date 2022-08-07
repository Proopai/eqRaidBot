package bot

import (
	"eqRaidBot/db/model"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	attendStateChar  = 0
	attendStateEvent = 1
	attendStateDone  = 3
)

type AttendanceProvider struct {
	pool     *pgxpool.Pool
	registry map[string]AttendanceState
}

type AttendanceState struct {
	characterId int64
	eventId     int64
	state       int
	userId      string
}

func NewAttendanceProvider() *AttendanceProvider {
	return &AttendanceProvider{}
}

func (r *AttendanceProvider) recordAttendance(s *discordgo.Session, m *discordgo.MessageCreate) {
	// look up the event
	// mark this perosn as attending
}

func (r *AttendanceProvider) attendanceStep(s *discordgo.Session, m *discordgo.MessageCreate) {
	channel, err := s.UserChannelCreate(m.Author.ID)
	if err != nil {
		log.Print(err.Error())
		return
	}

	if err = sendMessage(s, channel, fmt.Sprintf("Great! See you at %s.\n", c)); err != nil {
		log.Println(err.Error())
	}
}

func (r *AttendanceProvider) init(c *discordgo.Channel, s *discordgo.Session, m *discordgo.MessageCreate) bool {
	// check the database to see if they have previously registered
	if _, ok := r.registry[m.Author.ID]; !ok {
		char := model.Character{}
		toons, err := char.GetByOwner(r.pool, m.Author.ID)
		if err != nil {
			log.Println(err.Error())
			_ = sendMessage(s, c, "There was an error with your input - please try again")
		}

		if len(toons) == 0 {
			if err := sendMessage(s, c, fmt.Sprintf("You have no characters register, please type **!register** to add one.")); err != nil {
				return false
			}
		}

		r.registry[m.Author.ID] = AttendanceState{
			state:  attendStateChar,
			userId: m.Author.ID,
		}

		var charString []string

		for i, t := range toons {
			charString = append(charString, fmt.Sprintf("%d. %s", i, t.Name))
		}

		if err := sendMessage(s, c, fmt.Sprintf("Hello %s, which character will you be brining?\n%s", m.Author.Username, strings.Join(charString, "\n"))); err != nil {
			return false
		}

		return true
	}

	return false
}
