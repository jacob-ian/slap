package slap

import (
	"encoding/json"
	"net/http"

	"github.com/slack-go/slack"
)

// The payload of a Slack slash command request
type CommandPayload struct {
	// Deprecated: The verification token.
	Token string `json:"token"`
	// The command that was called
	Command string `json:"command"`
	// The text after the command
	Text string `json:"text"`
	// The Team ID of the workspace this command was used in.
	TeamID string `json:"team_id"`
	// The domain name of the workspace.
	TeamDomain string `json:"team_domain"`
	// The Enterprise ID this workspace belongs to if using Enterprise Grid.
	EnterpriseID string `json:"enterprise_id,omitempty"`
	// The name of the enterprise this workspace belongs to if using Enterprise Grid..
	EnterpriseName string `json:"enterprise_name,omitempty"`
	// The ID of the channel the command was used in.
	ChannelID string `json:"channel_id"`
	// The name of the channel the command was used in.
	ChannelName string `json:"channel_name"`
	// The ID of the user calling the command.
	UserID string `json:"user_id"`
	// Deprecated: The name of the user calling the command.
	UserName string `json:"user_name"`
	// A temporary webhook URL that used to generate message responses.
	ResponseURL string `json:"response_url"`
	// A short-lived ID that can be used to open modals.
	TriggerID string `json:"trigger_id"`
	// Your Slack App's unique identifier.
	APIAppID string `json:"api_app_id"`
}

// A slash command request from Slack
type CommandRequest struct {
	baseRequest
	// The paylaod of the Slack request
	Payload CommandPayload
}

// A function to handle slash command requests
type CommandHandler func(req *CommandRequest) error

// The type in a command action response
type CommandActionResponseType string

const (
	RespondInChannel CommandActionResponseType = "in_channel"
	RespondEphemeral CommandActionResponseType = "ephemeral"
)

// An immediate action to be ran in response to a slash command
type CommandAction struct {
	// The type of response
	ResponseType CommandActionResponseType `json:"response_type"`
	// Text to send in the response
	Text string `json:"text"`
	// Slack Block Kit Blocks to send in the response
	Blocks []slack.Block `json:"blocks,omitempty"`
}

// Immediately respond to Slack's slash command request with an action
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

func (app *Application) handleCommand(w http.ResponseWriter, r *http.Request) {
	var payload CommandPayload
	err := json.NewDecoder(r.Body).Decode(&payload)
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
		_, msgerr := req.Client.PostEphemeral(req.Payload.ChannelID, req.Payload.UserID, slack.MsgOptionText(app.errorMessage, false))
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
