package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/joho/godotenv"
	"github.com/olebedev/when"
	"github.com/olebedev/when/rules/common"
	"github.com/olebedev/when/rules/en"
	tb "gopkg.in/tucnak/telebot.v2"
)

type Reminder struct {
	Object     string
	RemindDate string
	SenderId   string
}

func insertReminder(reminder *Reminder) error {
	endpoint := fmt.Sprintf("%s/rest/v1/Reminder", os.Getenv("SUPABASE_URL"))

	payload, err := json.Marshal(&map[string]string{
		"object":      reminder.Object,
		"remind_date": reminder.RemindDate,
		"sender_id":   reminder.SenderId,
	})

	if err != nil {
		log.Fatalf("error while encoding reminder. %s", err)
	}

	client := &http.Client{Timeout: time.Second * 10}
	request, _ := http.NewRequest("POST", endpoint, bytes.NewBuffer(payload))

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("apikey", os.Getenv("SUPABASE_SECRET_KEY"))
	request.Header.Set("Authorization", "Bearer "+os.Getenv("SUPABASE_SECRET_KEY"))
	request.Header.Set("Prefer", "return=representation")

	response, _ := client.Do(request)

	if err != nil || response.StatusCode != 201 {
		log.Fatalf("error while POSTing reminder. %s", err)
	}

	return nil
}

func listPendingReminders() {
	//endpoint := fmt.Sprintf("%s/rest/v1/Reminder", os.Getenv("SUPABASE_URL"))
}

func main() {
	// loading .env file
	godotenv.Load()

	b, err := tb.NewBot(tb.Settings{
		Token:       os.Getenv("REMEMBR_TELEGRAM_BOT_TOKEN"),
		Synchronous: true,
	})

	if err != nil {
		log.Fatal(err)
		return
	}

	// initialising natural language parser
	w := when.New(nil)
	w.Add(en.All...)
	w.Add(common.All...)
	// TODO: get timezone from user preferences
	tz, _ := time.LoadLocation("Europe/Paris")

	b.Handle(tb.OnText, func(m *tb.Message) {
		now := time.Now().In(tz)
		r, err := w.Parse(m.Text, now)

		// an error has occurred
		if err != nil {
			b.Send(
				m.Sender,
				"Something wrong happened while handling your message, please try again later.",
			)
		}

		// no date matching was found in the message
		if r == nil || m.Text[0:r.Index] == "" {
			b.Send(m.Sender, "Invalid task and/or date format.")
			return
		}

		// given date in in the past
		if r.Time.Before(now) {
			b.Send(m.Sender, "Given date should be in the future.")
			return
		}

		reminder := Reminder{
			Object:     m.Text[0:r.Index],
			RemindDate: r.Time.Format(time.RFC3339), // 2006-01-02T15:04:05Z07:00
			SenderId:   fmt.Sprint(m.Sender.ID),
		}

		// inserting reminders in database
		err = insertReminder(&reminder)
		if err != nil {
			b.Send(
				m.Sender,
				"Something wrong happened while handling your message, please try again later.",
			)
		}

		b.Send(m.Sender, fmt.Sprintf(
			"✅ I will remind you \"%s\" on %s",
			reminder.Object,
			r.Time.Format(time.RFC850),
		))
	})

	lambda.Start(func(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
		if req.HTTPMethod == "GET" {
			// reminders :=

			return &events.APIGatewayProxyResponse{
				StatusCode: 200,
			}, nil
		} else if req.HTTPMethod == "POST" {
			var u tb.Update
			if err = json.Unmarshal([]byte(req.Body), &u); err == nil {
				b.ProcessUpdate(u)

				return &events.APIGatewayProxyResponse{
					StatusCode: 200,
				}, nil
			}

			return &events.APIGatewayProxyResponse{
				StatusCode: 400,
			}, nil
		}

		return &events.APIGatewayProxyResponse{
			StatusCode: 405,
		}, nil
	})
}
