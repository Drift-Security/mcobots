package mcobots

import (
	"errors"
	"time"

	"log"

	"github.com/stroncium/discordgo"
)

type StatusBotUpdater func(*StatusBot) error

type StatusBot struct {
	Token    string
	Interval time.Duration
	Updater  StatusBotUpdater

	LastUpdateSuccessTime time.Time

	Session *discordgo.Session
	close   chan interface{}
	timer   *time.Timer
}

func (bot *StatusBot) RunUpdateCycle() {
	err := bot.Updater(bot)
	if err == nil {
		bot.LastUpdateSuccessTime = time.Now()
	} else {
		since := int(bot.LastUpdateSuccessTime.Unix())
		bot.Session.UpdateStatusComplex(discordgo.UpdateStatusData{
			Status:    "idle",
			IdleSince: &since,
			Game: &discordgo.Game{
				Name: "with broken bytes",
				Type: discordgo.GameTypeGame,
			},
		})
		log.Printf("error updating status: %v", err)
	}
}

func (bot *StatusBot) Start() (err error) {
	log.Printf("starting bot")
	Session, err := discordgo.New(bot.Token)
	if err != nil {
		return
	}
	bot.Session = Session
	// bot.Session.AddHandler(onMessage)
	err = bot.Session.Open()
	if err != nil {
		return
	}
	bot.close = make(chan interface{})
	go bot.Run()
	return
}

func (bot *StatusBot) Stop() chan interface{} {
	log.Printf("stopping bot")
	bot.close <- nil
	return bot.close
}

func (bot *StatusBot) Run() {
	bot.timer = time.NewTimer(bot.Interval)
	bot.RunUpdateCycle()
	for {
		select {
		case <-bot.close:
			bot.timer.Stop()
			bot.Session.Close()
			bot.Session = nil
			bot.close <- nil
			return
		case <-bot.timer.C:
			bot.timer.Reset(bot.Interval)
			bot.RunUpdateCycle()
		}
	}
}

func (bot *StatusBot) UpdateStatus(upd discordgo.UpdateStatusData) (err error) {
	if err = AssertSensibleStatusUpdate(&upd); err != nil {
		return
	}
	if err = bot.Session.UpdateStatusComplex(upd); err != nil {
		return
	}
	return
}

func AssertSensibleStatusUpdate(upd *discordgo.UpdateStatusData) error {
	if upd.Game.Type != discordgo.GameTypePlaying && (upd.Game.TimeStamps.StartTimestamp != 0 || upd.Game.TimeStamps.EndTimestamp != 0) {
		return errors.New("timestamps only work when playing")
	}
	return nil
}

// func onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
// 	if m.Author.ID == s.State.User.ID {
// 		return
// 	}
// 	if m.Content == "ping" {
// 		s.MessageReactionAdd(m.ChannelID, m.ID, "🇵")
// 		s.MessageReactionAdd(m.ChannelID, m.ID, "🇴")
// 		s.MessageReactionAdd(m.ChannelID, m.ID, "🇳")
// 		s.MessageReactionAdd(m.ChannelID, m.ID, "🇬")
// 		// s.ChannelMessageSend(m.ChannelID, "Pong!")
// 		return
// 	}
// 	log.Printf("msg \"%s\" %x", m.Content, m.Content)
// }
