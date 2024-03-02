package slap

import (
	"encoding/json"
	"net/http"

	"github.com/slack-go/slack"
)

type CommandRequest struct {
	baseRequest
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
func (req *CommandRequest) AckWithAction(action CommandAction) {
	if req.ackCalled {
		return
	}
	req.ackCalled = true
	bytes, err := json.Marshal(action)
	if err != nil {
		req.Logger.Error("Could not encode command response action", "error", err.Error())
		req.errChannel <- err
		return
	}
	req.ackChannel <- bytes
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
		app.logger.Error("Could not get bot token", "teamID", payload.TeamID, "error", err.Error())
		http.Error(w, "An error occurred", http.StatusInternalServerError)
		return
	}

	ackChan := make(chan []byte)
	errChan := make(chan error)

	go func() {
		req := &CommandRequest{
			Payload: payload,
			baseRequest: baseRequest{
				errChannel: errChan,
				ackChannel: ackChan,
				ackCalled:  false,
				writer:     w,
				Client:     slack.New(botToken),
				Logger:     app.logger,
			},
		}
		err := handler(req)
		if err == nil {
			return
		}
		app.logger.Error("A command handler failed", "command", req.Payload.Command, "error", err.Error())
		_, msgerr := req.Client.PostEphemeral(req.Payload.ChannelID, req.Payload.UserID, slack.MsgOptionText("An error occurred", false))
		if msgerr != nil {
			app.logger.Error("Unable to send error message to user", "user", req.Payload.UserID, "error", msgerr.Error())
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
