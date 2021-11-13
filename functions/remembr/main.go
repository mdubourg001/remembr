package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/joho/godotenv"
	tb "gopkg.in/tucnak/telebot.v2"
)

func handler(request events.APIGatewayProxyRequest) (err error) {
	b, err := tb.NewBot(tb.Settings{
		Token:       os.Getenv("REMEMBR_TELEGRAM_BOT_TOKEN"),
		Synchronous: true,
	})

	if err != nil {
		log.Fatal(err)
		return
	}

	b.Handle("/hello", func(m *tb.Message) {
		b.Send(m.Sender, "Hello World!")
	})

	var u tb.Update
	if err = json.Unmarshal([]byte(request.Body), &u); err == nil {
		b.ProcessUpdate(u)
	}

	return
}

func main() {
	// loading .env file
	godotenv.Load()

	// Make the handler available for Remote Procedure Call by AWS Lambda
	lambda.Start(handler)
}
