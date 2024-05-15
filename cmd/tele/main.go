package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"ytstalker/cmd/app/handlers"

	"github.com/NicoNex/echotron/v3"
	"zombiezen.com/go/sqlite/sqlitex"
)

type Session struct {
	MessageID int
	Text string
	Year int
}

const YouTubeFounded = 2006

const DefaultYear = 2006

func main() {
	token := os.Getenv("TG_TOKEN")
	if token == "" {
		log.Fatal("no telegram token provided")
	}

	dsn := os.Getenv("DSN")
	if dsn == "" {
		log.Fatal("no dsn provided")
	}
	
	db, err := sqlitex.NewPool(dsn, sqlitex.PoolOptions{PoolSize: 100})
	if err != nil {
		log.Fatal("cannot open db", err)
	}

	api := echotron.NewAPI(token)
	for u := range echotron.PollingUpdates(token) {

		visitor :=  strconv.FormatInt(u.ChatID(), 10)

		if u.Message != nil && u.Message.Text == "/start" {

			conn := db.Get(context.Background())
			sc := &handlers.SearchCriteria{
				YearsFrom: strconv.Itoa(DefaultYear),
				YearsTo: strconv.Itoa(DefaultYear),
			}
			video, err := handlers.TakeFirstUnseen(conn, visitor, sc)
			db.Put(conn)
			if err != nil {
				log.Println("cannot take video:", err)
				continue
			}


			text := "Title: " + video.Title + "\n" +
					"Views: " + strconv.FormatInt(video.Views, 10) + "\n" +
					"Uploaded: " + time.Unix(video.UploadedAt, 0).Format("02.01.2006") + "\n" +
					"\n" + 
					"https://www.youtube.com/watch?v=" + video.ID


			markup := GetKeyboard(DefaultYear)
			_, err = api.SendMessage(text, u.ChatID(), &echotron.MessageOptions{ReplyMarkup: markup})
			if err != nil {
				fmt.Println("cannot send message:", err)
				continue
			}

			err = handlers.RememberSeen(conn, visitor, video.ID)
			if err != nil {
				log.Println("cannot remember seen: ", err)
			}
		}

		if u.CallbackQuery != nil && strings.HasPrefix(u.CallbackQuery.Data, "/random") {

			year, err := strconv.Atoi(strings.Replace(u.CallbackQuery.Data, "/random", "", 1))
			if err != nil {
				log.Println("cannot parse year:", err)
				continue
			}

			conn := db.Get(context.Background())
			sc := &handlers.SearchCriteria{
				YearsFrom: strconv.Itoa(year),
				YearsTo: strconv.Itoa(year),
			}
			video, err := handlers.TakeFirstUnseen(conn, visitor, sc)
			db.Put(conn)
			if err != nil {
				log.Println("cannot take video:", err)
				continue
			}

			kb := u.CallbackQuery.Message.ReplyMarkup

			prevMsg := echotron.NewMessageID(u.ChatID(), u.CallbackQuery.Message.ID)
			_, err = api.EditMessageText(u.CallbackQuery.Message.Text, prevMsg, nil)
			if err != nil {
				log.Println("cannot remove keyboard from prev message:", err)
			}
			
			text := "Title: " + video.Title + "\n" +
					"Views: " + strconv.FormatInt(video.Views, 10) + "\n" +
					"Uploaded: " + time.Unix(video.UploadedAt, 0).Format("02.01.2006") + "\n" +
					"\n" + 
					"https://www.youtube.com/watch?v=" + video.ID

			_, err = api.SendMessage(text, u.ChatID(), &echotron.MessageOptions{ReplyMarkup: kb})
			if err != nil {
				fmt.Println("cannot send message:", err)
				continue
			}

			err = handlers.RememberSeen(conn, visitor, video.ID)
			if err != nil {
				log.Println("cannot remember seen: ", err)
			}

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

		}
		
	}
}

var ErrNoSession = errors.New("no session found")


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