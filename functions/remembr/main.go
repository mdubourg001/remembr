package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/joho/godotenv"
	"github.com/olebedev/when"
	"github.com/olebedev/when/rules/common"
	"github.com/olebedev/when/rules/en"
	tb "gopkg.in/tucnak/telebot.v2"
)

type ApiReminder struct {
	Id          int
	Created_at  string
	Object      string
	Remind_date string
	Sender_id   int
}

type Reminder struct {
	Object     string
	RemindDate string
	SenderId   string
}

func GetPreparedHttpClient(
	verb string,
	body *bytes.Buffer,
	queryParamsStr string,
) (*http.Client, *http.Request) {
	endpoint := fmt.Sprintf("%s/rest/v1/Reminder%s", os.Getenv("SUPABASE_URL"), queryParamsStr)

	client := &http.Client{Timeout: time.Second * 10}
	request, _ := http.NewRequest(verb, endpoint, body)

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("apikey", os.Getenv("SUPABASE_SECRET_KEY"))
	request.Header.Set("Authorization", "Bearer "+os.Getenv("SUPABASE_SECRET_KEY"))
	request.Header.Set("Prefer", "return=representation")

	return client, request
}

func ForgePendingReminderMessage(object string) string {
	return fmt.Sprintf("📣 %s", object)
}

func ParsePendingReminderMessage(message string) string {
	messageOnly := strings.Fields(message)[1:]
	return strings.Join(messageOnly, " ")
}

func CreateReminder(object string, remindDate time.Time, senderID int) Reminder {
	return Reminder{
		Object:     object,
		RemindDate: remindDate.Format(time.RFC3339), // 2006-01-02T15:04:05Z07:00
		SenderId:   fmt.Sprint(senderID),
	}
}

func InsertReminder(reminder *Reminder) error {
	log.Printf("Creating new reminder: %+v", reminder)

	payload, err := json.Marshal(&map[string]string{
		"object":      reminder.Object,
		"remind_date": reminder.RemindDate,
		"sender_id":   reminder.SenderId,
	})

	if err != nil {
		log.Fatalf("error while encoding reminder. %s", err)
	}

	client, request := GetPreparedHttpClient("POST", bytes.NewBuffer(payload), "")
	log.Printf("Sending POST request to create reminder")
	response, err := client.Do(request)

	if err != nil || response.StatusCode != 201 {
		log.Fatalf("error while POSTing reminder. %s", err)
	}

	log.Printf("Successfully created reminder with status code: %d", response.StatusCode)
	return err
}

func ListPendingReminders() (*[]ApiReminder, error) {
	tz, _ := time.LoadLocation("Europe/Paris")
	now := time.Now().In(tz)

	log.Printf("Fetching pending reminders for time: %s", now.Format(time.RFC3339))

	client, request := GetPreparedHttpClient(
		"GET",
		bytes.NewBuffer([]byte{}),
		fmt.Sprintf("?remind_date=lte.%s", url.QueryEscape(now.Format(time.RFC3339))),
	)
	response, err := client.Do(request)

	if err != nil || response.StatusCode != 200 {
		log.Fatalf("error while listing pending reminders. %s", err)
	}

	defer response.Body.Close()
	bodyBytes, _ := ioutil.ReadAll(response.Body)

	if err != nil {
		log.Fatal(err)
	}

	var reminders []ApiReminder
	err = json.Unmarshal(bodyBytes, &reminders)

	log.Printf("Found %d pending reminders", len(reminders))

	return &reminders, err
}

func DeletePassedReminders() error {
	tz, _ := time.LoadLocation("Europe/Paris")
	now := time.Now().In(tz)

	log.Printf("Deleting reminders before: %s", now.Format(time.RFC3339))

	client, request := GetPreparedHttpClient(
		"DELETE",
		bytes.NewBuffer([]byte{}),
		fmt.Sprintf("?remind_date=lte.%s", url.QueryEscape(now.Format(time.RFC3339))),
	)
	response, err := client.Do(request)

	if err != nil || response.StatusCode != 200 {
		log.Fatalf("error while deleting passed reminders. %s", err)
	}

	log.Printf("Successfully deleted passed reminders with status code: %d", response.StatusCode)
	return err
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

	// snooze buttons durations
	btns := [5]string{"5m", "20m", "1h", "3h", "1d"}

	btnsDurations := make(map[string]time.Duration)
	btnsDurations[btns[0]] = time.Minute * 5
	btnsDurations[btns[1]] = time.Minute * 20
	btnsDurations[btns[2]] = time.Hour * 1
	btnsDurations[btns[3]] = time.Hour * 3
	btnsDurations[btns[4]] = time.Hour * 24

	// initialising natural language parser
	w := when.New(nil)
	w.Add(en.All...)
	w.Add(common.All...)
	// TODO: get timezone from user preferences
	tz, _ := time.LoadLocation("Europe/Paris")

	// handler for inline keyboard buttons (snooze buttons)
	b.Handle(tb.OnCallback, func(c *tb.Callback) {
		data := strings.Split(c.Data, "|")
		object := data[1]
		btn := data[2]

		remindDate := time.Now().In(tz).Add(btnsDurations[btn])
		reminder := CreateReminder(object, remindDate, c.Sender.ID)
		err = InsertReminder(&reminder)
		if err != nil {
			b.Send(
				c.Sender,
				"❌ Something wrong happened while handling your message, please try again later.",
			)
			return
		}

		b.Send(c.Sender, fmt.Sprintf(
			"✅ I will remind you \"%s\" on %s",
			reminder.Object,
			remindDate,
		))

		b.Respond(c, &tb.CallbackResponse{})
	})

	// handler for regular reminder messages
	b.Handle(tb.OnText, func(m *tb.Message) {
		now := time.Now().In(tz)
		r, err := w.Parse(m.Text, now)

		// an error has occurred
		if err != nil {
			b.Send(
				m.Sender,
				"❌ Something wrong happened while handling your message, please try again later.",
			)
		}

		// no date matching was found in the message
		if r == nil || m.Text[0:r.Index] == "" {
			b.Send(m.Sender, "❌ Invalid task and/or date format.")
			return
		}

		// given date in in the past
		if r.Time.Before(now) {
			b.Send(m.Sender, "❌ Given date should be in the future.")
			return
		}

		// inserting reminders in database
		reminder := CreateReminder(m.Text[0:r.Index-1], r.Time, m.Sender.ID)
		err = InsertReminder(&reminder)
		if err != nil {
			b.Send(
				m.Sender,
				"❌ Something wrong happened while handling your message, please try again later.",
			)
			return
		}

		b.Send(m.Sender, fmt.Sprintf(
			"✅ I will remind you \"%s\" on %s",
			reminder.Object,
			r.Time.Format(time.RFC850),
		))
	})

	lambda.Start(func(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
		log.Printf("Received %s request", req.HTTPMethod)

		if req.HTTPMethod == "GET" {
			reminders, err := ListPendingReminders()

			if err == nil {
				log.Printf("Processing %d pending reminders", len(*reminders))
				for _, reminder := range *reminders {
					snoozeKb := &tb.ReplyMarkup{}

					row := make([]tb.Btn, 0)
					for _, strDur := range btns {
						btn := snoozeKb.Data(
							strDur,
							fmt.Sprintf("%d_%s", reminder.Id, strDur),
							reminder.Object,
							strDur,
						)

						row = append(row, btn)
					}

					snoozeKb.Inline(row)

					reminderMessage := ForgePendingReminderMessage(reminder.Object)
					b.Send(&tb.User{ID: reminder.Sender_id}, reminderMessage, snoozeKb)
				}
			}

			if len(*reminders) > 0 {
				err = DeletePassedReminders()
			}

			if err == nil {
				return &events.APIGatewayProxyResponse{
					StatusCode: 200,
				}, nil
			}

			log.Printf("Error processing reminders: %s", err)
			return &events.APIGatewayProxyResponse{
				StatusCode: 500,
				Body:       err.Error(),
			}, nil
		} else if req.HTTPMethod == "POST" {
			log.Printf("Processing webhook update")
			var u tb.Update
			if err = json.Unmarshal([]byte(req.Body), &u); err == nil {
				b.ProcessUpdate(u)
				log.Printf("Successfully processed webhook update")
				return &events.APIGatewayProxyResponse{
					StatusCode: 200,
				}, nil
			}

			log.Printf("Failed to process webhook update: %s", err)
			return &events.APIGatewayProxyResponse{
				StatusCode: 400,
			}, nil
		}

		log.Printf("Method not allowed: %s", req.HTTPMethod)
		return &events.APIGatewayProxyResponse{
			StatusCode: 405,
		}, nil
	})
}
