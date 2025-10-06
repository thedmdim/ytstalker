package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"ytstalker/cmd/app/handlers"
	"github.com/NicoNex/echotron/v3"
)

const YouTubeFounded = 2006
const DefaultYear = 2006
const WebhookURL = "https://ytstalker.mov/webhook/telegram"
var   apiURL = os.Getenv("API_URL") // localhost/api
var   token = os.Getenv("TG_TOKEN")

func main() {
	if token == "" {
		log.Fatal("no telegram token provided")
	}

	api := echotron.NewAPI(token)
	res, err := api.GetMe()
	if err != nil {
		log.Fatal("cannot get my username", err)
	}
	me := res.Result.Username

	for u := range echotron.PollingUpdates(token) {

		if u.Message != nil && (u.Message.Text == "/start" || u.Message.Text == "/start@"+me ) {

			data, err := RequestAPI(u.ChatID(), DefaultYear)
			if err != nil {
				fmt.Println(err)
				continue
			}

			text := MakeText(data)

			markup := GetKeyboard(DefaultYear)
			_, err = api.SendMessage(text, u.ChatID(), &echotron.MessageOptions{ReplyMarkup: markup})
			if err != nil {
				fmt.Println("cannot send message:", err)
				continue
			}

			continue
		}

		if u.CallbackQuery != nil && strings.HasPrefix(u.CallbackQuery.Data, "/random") {

			year, err := strconv.Atoi(strings.Replace(u.CallbackQuery.Data, "/random", "", 1))
			if err != nil {
				log.Println("cannot parse year:", err)
				continue
			}

			data, err := RequestAPI(u.ChatID(), year)
			if err != nil {
				fmt.Println(err)
				continue
			}

			kb := u.CallbackQuery.Message.ReplyMarkup

			prevMsg := echotron.NewMessageID(u.ChatID(), u.CallbackQuery.Message.ID)
			_, err = api.EditMessageText(u.CallbackQuery.Message.Text, prevMsg, nil)
			if err != nil {
				log.Println("cannot remove keyboard from prev message:", err)
			}
			
			text := MakeText(data)

			_, err = api.SendMessage(text, u.ChatID(), &echotron.MessageOptions{ReplyMarkup: kb})
			if err != nil {
				fmt.Println("cannot send message:", err)
				continue
			}

			continue
		}


		if u.CallbackQuery != nil && strings.HasPrefix(u.CallbackQuery.Data, "/set") {

			year, err := strconv.Atoi(strings.Replace(u.CallbackQuery.Data, "/set", "", 1))
			if err != nil {
				log.Println("cannot parse year:", err)
				continue
			}

			kb := GetKeyboard(year)
			_, err = api.EditMessageText(
				u.CallbackQuery.Message.Text,
				echotron.NewMessageID(u.ChatID(), u.CallbackQuery.Message.ID),
				&echotron.MessageTextOptions{ReplyMarkup: kb},
			)

			if err != nil {
				fmt.Println("cannot set year:", err)
				continue
			}
			continue
		}
		
	}
}

func GetKeyboard(year int) echotron.InlineKeyboardMarkup {

	firstRow := make([]echotron.InlineKeyboardButton, 0, 3)
	if year <= YouTubeFounded {
		firstRow = append(firstRow, echotron.InlineKeyboardButton{Text: "X", CallbackData: "/noaction"})
	} else {
		firstRow = append(firstRow, echotron.InlineKeyboardButton{Text: "<-", CallbackData: "/set" + strconv.Itoa(year-1)})
	}

	firstRow = append(firstRow, echotron.InlineKeyboardButton{Text: strconv.Itoa(year), CallbackData: "/noaction" + strconv.Itoa(year)})

	if year >= time.Now().Year() {
		firstRow = append(firstRow, echotron.InlineKeyboardButton{Text: "X", CallbackData: "/noaction"})
	} else {
		firstRow = append(firstRow, echotron.InlineKeyboardButton{Text: "->", CallbackData: "/set" + strconv.Itoa(year+1)})
	}
	
	secondRow := []echotron.InlineKeyboardButton{{Text: "Random", CallbackData: "/random" + strconv.Itoa(year)}}

	return echotron.InlineKeyboardMarkup{InlineKeyboard: [][]echotron.InlineKeyboardButton{firstRow, secondRow}}
}

func RequestAPI(visitor int64, year int) (handlers.VideoWithReactions, error) {
	data := handlers.VideoWithReactions{} 

	resp, err := http.Get(fmt.Sprintf("http://%s/videos/random?year=%d&visitor=%d", apiURL, year, visitor))
	if err != nil {
		return data, fmt.Errorf("cannot get random id: %w", err)
	}

	buff := bytes.Buffer{}
	buff.ReadFrom(resp.Body)
	resp.Body.Close()
	resp, err = http.Get(fmt.Sprintf("http://%s/videos/%s?visitor=%d", apiURL, buff.String(), visitor))
	if err != nil {
		return data, fmt.Errorf("cannot get video info: %w", err)
	}

	json.NewDecoder(resp.Body).Decode(&data)
	return data, nil
}

func MakeText(data handlers.VideoWithReactions) string {
	return "Title: " + data.Video.Title + "\n" +
			"Views: " + strconv.FormatInt(data.Video.Views, 10) + "\n" +
			"Uploaded: " + time.Unix(data.Video.UploadedAt, 0).Format("02.01.2006") + "\n" +
			"\n" + 
			"https://www.youtube.com/watch?v=" + data.Video.ID
}