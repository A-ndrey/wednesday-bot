package main

import (
	"encoding/json"
	"fmt"
	"gopkg.in/telebot.v3"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"
)

const recFileName = "recipients"

const phrase = "It is Wednesday, my dudes"

var recipients = make(map[RecID]struct{})
var mx sync.Mutex

type RecID string

func (r RecID) Recipient() string {
	return string(r)
}

func main() {
	rand.Seed(time.Now().Unix())

	pref := telebot.Settings{
		Token:  os.Getenv("TOKEN"),
		Poller: &telebot.LongPoller{Timeout: 60 * time.Second},
	}

	loadRecipients()

	b, err := telebot.NewBot(pref)
	if err != nil {
		log.Fatalln(err)
	}

	b.Handle("/start", func(ctx telebot.Context) error {
		rec := ctx.Recipient()
		if rec == nil || rec.Recipient() == "" {
			return nil
		}

		mx.Lock()
		defer mx.Unlock()

		if _, ok := recipients[RecID(rec.Recipient())]; !ok {
			recipients[RecID(rec.Recipient())] = struct{}{}
			saveRecipients()
		}

		return nil
	})

	go sendFrogs(b)

	b.Start()
}

func loadRecipients() {
	data, err := os.ReadFile(recFileName)
	if err != nil {
		log.Println(err)
		return
	}

	err = json.Unmarshal(data, &recipients)
	if err != nil {
		log.Println(err)
		return
	}
}

func saveRecipients() {
	data, err := json.Marshal(&recipients)
	if err != nil {
		log.Println(err)
		return
	}

	err = os.WriteFile(recFileName, data, 0666)
	if err != nil {
		log.Println(err)
		return
	}
}

func sendFrogs(bot *telebot.Bot) {
	for {
		nextST := nextSendTime()
		fmt.Println("Next send time: ", time.Now().Add(nextST).String())
		time.Sleep(nextST)

		filename, err := randomFile()
		if err != nil {
			log.Println(err)
			continue
		}

		f := telebot.Photo{
			File:    telebot.FromDisk(filename),
			Caption: phrase,
		}

		for rec := range recipients {
			_, err = bot.Send(rec, &f)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func nextSendTime() time.Duration {
	const sendHour = 6

	now := time.Now().In(time.UTC)
	day := time.Date(now.Year(), now.Month(), now.Day(), sendHour, 0, 0, 0, time.UTC)
	weekDur := time.Wednesday - day.Weekday()
	if weekDur > 0 {
		day = day.Add(time.Duration(weekDur) * 24 * time.Hour)
	} else if weekDur < 0 {
		day = day.Add((time.Duration(weekDur) + 7) * 24 * time.Hour)
	} else if now.Hour() >= sendHour {
		day = day.Add(7 * 24 * time.Hour)
	}

	return day.Sub(now)
}

func randomFile() (string, error) {
	const frogsDir = "resources/frogs"

	dir, err := os.ReadDir(frogsDir)
	if err != nil {
		return "", err
	}

	frogNum := rand.Intn(len(dir))

	return frogsDir + "/" + dir[frogNum].Name(), nil
}
