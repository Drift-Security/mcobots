package main

import (
	"errors"
	"github.com/stroncium/discordgo"
	"github.com/stroncium/mcobots"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const MCO_LIFE_ENDPOINT = "https://mco-life-api.herokuapp.com/status"

type Reservations struct {
	Reservations struct {
		Today uint `json:"today"`
		Total uint `json:"total"`
	} `json:"reservations"`
}

func getMCOReservations() (res Reservations, err error) {
	var req *http.Request
	if req, err = http.NewRequest("GET", MCO_LIFE_ENDPOINT, nil); err != nil {
		return
	}

	req.Header.Set("Accept", "*/*")

	var resp *http.Response
	resp, err = (&http.Client{}).Do(req)
	if err != nil {
		return
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(bytes, &res)
	if err != nil {
		return
	}
	return
}

// const ROCKET_EMOJI = "🚀"

func MCOReservationsUpdater(bot *mcobots.StatusBot) (err error) {
	reservs, err := getMCOReservations()
	if err != nil {
		return
	}

	name := fmt.Sprintf("%d today", reservs.Reservations.Today)
	details := fmt.Sprintf("%d total", reservs.Reservations.Total)

	upd := discordgo.UpdateStatusData{
		AFK:    false,
		Status: "online",
		Game: &discordgo.Game{
			Name:    name,
			Type:    discordgo.GameTypeListening,
			Details: details,
		},
	}
	if err = bot.UpdateStatus(upd); err != nil {
		return
	}
	return nil
}

const TOKEN_ENV_VAR = "MCOBOT_RESERVS_TOKEN"

func main() {
	var err error

	token := os.Getenv(TOKEN_ENV_VAR)
	if token == "" {
		panic(fmt.Errorf("Environment %s required", TOKEN_ENV_VAR))
	}

	bot := &mcobots.StatusBot{
		Token:    token,
		Interval: time.Second * 60,
		Updater:  MCOReservationsUpdater,
	}

	if err = bot.Start(); err != nil {
		panic(err)
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt, syscall.SIGTERM)
	<-sc
	log.Printf("shutting down...")

	done := make(chan error)
	go func() {
		select {
		case <-sc:
			done <- errors.New("forced shutdown")
		case <-bot.Stop():
		}
		done <- nil
	}()
	err = <-done
	if err != nil {
		log.Printf("shutdown error: %v", err)
	} else {
		log.Printf("shutdown done")
	}
}
