package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/jacob-ian/slap"
	"github.com/slack-go/slack"
)

func main() {
	router := http.NewServeMux()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	app := slap.New(slap.Config{
		Router:        router,
		SigningSecret: os.Getenv("SIGNING_SECRET"),
		BotToken: func(teamID string) (string, error) {
			return os.Getenv("BOT_TOKEN"), nil
		},
		Logger: logger,
	})

	app.RegisterCommand("/start", func(req *slap.CommandRequest) error {
		req.AckWithAction(slap.CommandResponseAction{
			ResponseType: slap.RespondEphemeral,
			Text:         "Get started by clicking the button!",
			Blocks: []slack.Block{
				slack.SectionBlock{
					Type: "section",
					Text: &slack.TextBlockObject{
						Type: "plain_text",
						Text: "Get started by clicking the button!",
					},
					Accessory: &slack.Accessory{
						ButtonElement: &slack.ButtonBlockElement{
							Type:     "button",
							ActionID: "start-button",
							Text: &slack.TextBlockObject{
								Type: "plain_text",
								Text: "Click Me!",
							},
						},
					},
				},
			},
		})
		return nil
	})

	app.RegisterBlockAction("start-button", func(req *slap.BlockActionRequest) error {
		req.Ack()

		_, err := req.Client.OpenView(req.Payload.TriggerID, slack.ModalViewRequest{
			Type:       "modal",
			CallbackID: "form-modal",
			Title: &slack.TextBlockObject{
				Type: "plain_text",
				Text: "Form",
			},
			Submit: &slack.TextBlockObject{
				Type: "plain_text",
				Text: "Submit",
			},
			Close: &slack.TextBlockObject{
				Type: "plain_text",
				Text: "Cancel",
			},
			Blocks: slack.Blocks{
				BlockSet: []slack.Block{
					slack.SectionBlock{
						Type: "section",
						Text: &slack.TextBlockObject{
							Type: "mrkdwn",
							Text: "*Welcome to the form!*\nPlease fill out your name below.",
						},
					},
					slack.InputBlock{
						Type:    "input",
						BlockID: "input-block",
						Label: &slack.TextBlockObject{
							Type: "plain_text",
							Text: "Full Name",
						},
						Element: slack.PlainTextInputBlockElement{
							Type:     "plain_text_input",
							ActionID: "full-name",
							Placeholder: &slack.TextBlockObject{
								Type: "plain_text",
								Text: "Enter your name",
							},
						},
					},
				},
			},
		})
		if err != nil {
			return err
		}

		return nil
	})

	app.RegisterViewSubmission("form-modal", func(req *slap.ViewSubmissionRequest) error {
		fullName, hasFullName := req.Payload.View.State.Values["input-block"]["full-name"]
		if !hasFullName {
			return errors.New("Missing full name")
		}

		if !startsWithA(fullName.Value) {
			req.AckWithAction(slap.ViewResponseAction{
				ResponseAction: slap.ViewResponseErrors,
				Errors: map[string]string{
					"input-block": "Name doesn't start with an 'A'",
				},
			})
			return nil
		}

		req.AckWithAction(slap.ViewResponseAction{
			ResponseAction: slap.ViewResponseClear,
		})

		_, _, err := req.Client.PostMessage(req.Payload.User.ID, slack.MsgOptionText("Hello "+fullName.Value, false))
		if err != nil {
			return err
		}

		return nil
	})

	app.RegisterEventHandler("message", func(req *slap.EventRequest) error {
		// Parse the inner message event from the Events API outer event
		var innerEvent slap.MessageEvent
		err := json.Unmarshal(req.Payload.Event, &innerEvent)
		if err != nil {
			return err
		}
		req.Ack()

		if innerEvent.IsBot() {
			// Slack will send a "message" event when the bot sends a message to a
			// conversation it is a part of
			return nil
		}
		_, _, err = req.Client.PostMessage(innerEvent.Channel, slack.MsgOptionText("You wrote: "+innerEvent.Text, false))
		if err != nil {
			return err
		}

		return nil
	})

	server := &http.Server{
		Addr:    "0.0.0.0:4000",
		Handler: router,
	}
	panic(server.ListenAndServe())
}

func startsWithA(value string) bool {
	return rune(strings.ToLower(value)[0]) != 'a'
}
