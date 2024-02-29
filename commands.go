package slap

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/slack-go/slack"
)

type CommandRequest struct {
	baseEvent
	Payload slack.SlashCommand
}

type CommandHandler func(req *CommandRequest) error

type CommandActionResponseType string

const (
	CmdRespondInChannel CommandActionResponseType = "in_channel"
	CmdRespondEphemeral CommandActionResponseType = "ephemeral"
)

type CommandAction struct {
	ResponseType CommandActionResponseType `json:"response_type"`
	Text         string                    `json:"text"`
	Blocks       []slack.Block             `json:"blocks,omitempty"`
}

// Immediately respond to Slack's Command request with an action
func (event *CommandRequest) AckWithAction(action CommandAction) {
	if event.ackCalled {
		return
	}
	event.ackCalled = true
	bytes, err := json.Marshal(action)
	if err != nil {
		slog.Error("Could not encode command response action", "error", err.Error())
		event.errChannel <- err
		return
	}
	event.ackChannel <- bytes
}

func (app *SlackApplication) handleCommand(w http.ResponseWriter, r *http.Request) {
	payload, err := slack.SlashCommandParse(r)
	if err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	handler, ok := app.commands[payload.Command]
	if !ok {
		http.Error(w, "Invalid command", http.StatusBadRequest)
		return
	}

	botToken, err := app.botToken(payload.TeamID)
	if err != nil {
		slog.Error("Could not find bot token", "teamID", payload.TeamID, "error", err.Error())
		http.Error(w, "An error occurred", http.StatusInternalServerError)
		return
	}

	ackChan := make(chan []byte)
	errChan := make(chan error)

	go func() {
		req := &CommandRequest{
			Payload: payload,
			baseEvent: baseEvent{
				errChannel: errChan,
				ackChannel: ackChan,
				ackCalled:  false,
				writer:     w,
				Client:     slack.New(botToken),
			},
		}
		err := handler(req)
		if err == nil {
			return
		}
		slog.Error("A command handler failed", "command", req.Payload.Command, "error", err.Error())
		_, msgerr := req.Client.PostEphemeral(req.Payload.ChannelID, req.Payload.UserID, slack.MsgOptionText("An error occurred", false))
		if msgerr != nil {
			slog.Error("Unable to send error message to user", "user", req.Payload.UserID, "error", msgerr.Error())
		}
		errChan <- err
	}()

	select {
	case ack := <-ackChan:
		if ack != nil {
			w.Header().Set("content-type", "application/json")
		}
		w.Write(ack)
	case err := <-errChan:
		if err == nil {
			return
		}
		http.Error(w, "An error occurred", http.StatusInternalServerError)
		return
	}
}
