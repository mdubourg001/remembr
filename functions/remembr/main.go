package main

import (
	"encoding/json"
	"fmt"
	"log"
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
		r, err := w.Parse(m.Text, time.Now().In(tz))

		// an error has occurred
		if err != nil {
			b.Send(
				m.Sender,
				"Something wrong happened while handling your message, please try again later.",
			)
		}

		// no date matching was found in the message
		if r == nil {
			b.Send(m.Sender, "Invalid task and/or date format.")
			return
		}

		// TODO: store reminder in database

		todo := m.Text[0:r.Index]
		b.Send(m.Sender, fmt.Sprintf(
			"✅ I will remind you \"%s\" on %s",
			todo,
			// TODO: format this prettier
			r.Time.String(),
		))
	})

	// Make the handler available for Remote Procedure Call by AWS Lambda
	lambda.Start(func(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
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
	})
}
