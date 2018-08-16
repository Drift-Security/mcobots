package main

import (
	"errors"
	"fmt"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	"log"

	"github.com/pdepip/go-binance/binance"
	"github.com/stroncium/discordgo"
	"github.com/stroncium/gg"
	"github.com/Drift-Security/mcobots"
)

type BinancePriceUpdater struct {
	currentAvatarStr string
}

type mcoPrices struct {
	mco24h binance.ChangeStats
	btcusd binance.TickerPrice
}

func (updater *BinancePriceUpdater) getPrices() (prices mcoPrices, err error) {
	client := binance.New("", "")
	mco24h, err := client.Get24Hr(binance.SymbolQuery{Symbol: "MCOBTC"})
	if err != nil {
		return
	}
	btcusd, err := client.GetLastPrice(binance.SymbolQuery{Symbol: "BTCUSDT"})
	if err != nil {
		return
	}
	prices.mco24h = mco24h
	prices.btcusd = btcusd
	return
}

func (updater *BinancePriceUpdater) tryUpdateAvatarIfNeeded(session *discordgo.Session, str string) (done chan error) {
	done = make(chan error, 1)
	if str == updater.currentAvatarStr {
		done <- nil
	} else {
		log.Printf("updating avatar to \"%s\"", str)
		go func() {
			img, err := drawCheapMoczAvatar(str)
			if err != nil {
				done <- err
				return
			}
			err = DiscordSetAvatar(session, img)
			if err != nil {
				done <- err
				return
			}
			updater.currentAvatarStr = str
			done <- nil
		}()
	}
	return
}

func (updater *BinancePriceUpdater) Updater(bot *mcobots.StatusBot) (err error) {
	startTime := time.Now()
	prices, err := updater.getPrices()

	avatarStr := fmt.Sprintf("$%.0f", math.Round(prices.mco24h.LastPrice*prices.btcusd.Price))
	avatarUpdateDone := updater.tryUpdateAvatarIfNeeded(bot.Session, avatarStr)

	usdPriceRounded := math.Round(100*prices.mco24h.LastPrice*prices.btcusd.Price) * 0.01
	name := fmt.Sprintf("$%.2f, %.1fksat MCO", usdPriceRounded, 100000.0*prices.mco24h.LastPrice)
	details := fmt.Sprintf("24h Change: %+0.2f%%", prices.mco24h.PriceChangePercent)
	state := fmt.Sprintf("Volume: %.1f BTC", prices.mco24h.Volume*prices.mco24h.WeightedAvgPrice)

	upd := discordgo.UpdateStatusData{
		AFK:    false,
		Status: "online",
		Game: &discordgo.Game{
			Name:    name,
			Type:    discordgo.GameTypePlaying,
			Details: details,
			TimeStamps: discordgo.TimeStamps{
				StartTimestamp: startTime.Unix(),
			},
			State: state,
		},
	}

	if err = bot.UpdateStatus(upd); err != nil {
		<-avatarUpdateDone
		return
	}
	err = <-avatarUpdateDone
	if err != nil {
		log.Printf("failed to update avatar: %v", err)
		return nil
	}
	return nil
}

func DiscordSetAvatar(s *discordgo.Session, img *gg.Context) (err error) {
	data, err := DiscordPNGData(img)
	if err != nil {
		return
	}
	if _, err = s.UserUpdate("", "", "", data, ""); err != nil {
		return
	}
	return
}

var fontBytes []byte = mcobots.MustAsset("fonts/Aller_Bd.ttf")

func drawCheapMoczAvatar(str string) (dc *gg.Context, err error) {
	w := 128.0
	h := 128.0
	dc = gg.NewContext(int(w), int(h))
	fontSize := 96.0
	if len(str) >= 2 {
		fontSize = 80.0
	}
	if len(str) >= 3 {
		fontSize = 64.0
	}
	if len(str) >= 4 {
		fontSize = 48.0
	}

	if err = dc.LoadFontFaceBytes(fontBytes, fontSize); err != nil {
		return
	}
	dc.SetRGB255(0x01, 0x26, 0x5A)
	dc.Clear()
	dc.SetRGB255(0xFF, 0xFF, 0xFF)
	dc.DrawStringAnchored(str, 64.0, 56.0, 0.5, 0.5)
	return
}

const TOKEN_ENV_VAR = "MCOBOT_PRICE_TOKEN"

func main() {
	var err error

	token := os.Getenv(TOKEN_ENV_VAR)
	if token == "" {
		panic(fmt.Errorf("Environment %s required", TOKEN_ENV_VAR))
	}

	binanceUpdater := BinancePriceUpdater{}

	bot := &mcobots.StatusBot{
		Token:    token,
		Interval: time.Second * 60,
		Updater:  binanceUpdater.Updater,
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
