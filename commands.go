package slap

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/slack-go/slack"
)

// The payload of a Slack slash command request
type CommandPayload struct {
	// Deprecated: The verification token.
	Token string
	// The command that was called
	Command string
	// The text after the command
	Text string
	// The Team ID of the workspace this command was used in.
	TeamID string
	// The domain name of the workspace.
	TeamDomain string
	// The Enterprise ID this workspace belongs to if using Enterprise Grid.
	EnterpriseID string
	// The name of the enterprise this workspace belongs to if using Enterprise Grid..
	EnterpriseName string
	// The ID of the channel the command was used in.
	ChannelID string
	// The name of the channel the command was used in.
	ChannelName string
	// The ID of the user calling the command.
	UserID string
	// Deprecated: The name of the user calling the command.
	UserName string
	// A temporary webhook URL that used to generate message responses.
	ResponseURL string
	// A short-lived ID that can be used to open modals.
	TriggerID string
	// Your Slack App's unique identifier.
	APIAppID string
}

func (p *CommandPayload) validate() error {
	if p.Command == "" || p.TeamID == "" || p.ChannelID == "" || p.UserID == "" || p.TriggerID == "" {
		return errors.New("Missing required value")
	}
	return nil
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
type CommandResponseActionType string

// The allowed CommandResponseAction ResponseType values
const (
	RespondInChannel CommandResponseActionType = "in_channel"
	RespondEphemeral CommandResponseActionType = "ephemeral"
)

// An immediate action to be ran in response to a slash command
type CommandResponseAction struct {
	// The type of response: "in_channel" or "ephemeral"
	ResponseType CommandResponseActionType `json:"response_type"`
	// Text to send in the response
	Text string `json:"text"`
	// Slack Block Kit Blocks to send in the response
	Blocks []slack.Block `json:"blocks,omitempty"`
}

// Immediately respond to Slack's slash command request with an action
func (req *CommandRequest) AckWithAction(action CommandResponseAction) {
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
	err := r.ParseForm()
	if err != nil {
		app.logger.Error("Failed to parse command request", "error", err.Error())
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	payload := CommandPayload{
		Token:          r.PostForm.Get("token"),
		Command:        r.PostForm.Get("command"),
		Text:           r.PostForm.Get("text"),
		TeamID:         r.PostForm.Get("team_id"),
		TeamDomain:     r.PostForm.Get("team_domain"),
		EnterpriseID:   r.PostForm.Get("enterprise_id"),
		EnterpriseName: r.PostForm.Get("enterprise_name"),
		ChannelID:      r.PostForm.Get("channel_id"),
		ChannelName:    r.PostForm.Get("channel_name"),
		UserID:         r.PostForm.Get("user_id"),
		UserName:       r.PostForm.Get("user_name"),
		ResponseURL:    r.PostForm.Get("response_url"),
		TriggerID:      r.PostForm.Get("trigger_id"),
		APIAppID:       r.PostForm.Get("api_app_id"),
	}

	if err = payload.validate(); err != nil {
		app.logger.Error("Command payload is invalid", "error", err.Error())
		http.Error(w, "Bad Request", http.StatusBadRequest)
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
