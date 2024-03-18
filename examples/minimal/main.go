package main

import (
	"net/http"
	"os"

	"github.com/jacob-ian/slap"
)

func main() {
	router := http.NewServeMux()

	app := slap.New(slap.Config{
		Router: router,
		BotToken: func(teamID string) (string, error) {
			return os.Getenv("BOT_TOKEN"), nil
		},
		SigningSecret: os.Getenv("SIGNING_SECRET"),
	})

	app.RegisterCommand("/start", func(req *slap.CommandRequest) error {
		req.AckWithAction(slap.CommandResponseAction{
			ResponseType: slap.RespondInChannel,
			Text:         "Hello world!",
		})
		return nil
	})

	server := &http.Server{
		Addr:    ":4000",
		Handler: router,
	}
	panic(server.ListenAndServe())
}
