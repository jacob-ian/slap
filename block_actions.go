package slap

import (
	"encoding/json"
	"net/http"

	"github.com/slack-go/slack"
)

// The payload of a Slack block action request
type BlockActionPayload struct {
	interactionPayload
	Container struct {
	} `json:"container"`
	Actions        []slack.BlockAction `json:"actions"`
	Hash           string              `json:"hash"`
	BotAccessToken string              `json:"bot_access_token"`
	Enterprise     *string             `json:"enterprise,omitempty"`
	Channel        *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"channel,omitempty"`
	Message *slack.MessageEvent      `json:"message,omitempty"`
	View    *slack.View              `json:"view,omitempty"`
	State   *slack.BlockActionStates `json:"state,omitempty"`
}

// A block action request
type BlockActionRequest struct {
	baseRequest
	Payload BlockActionPayload
}

// A function to handle a block action request
type BlockActionHandler func(req *BlockActionRequest) error

func (app *Application) handleBlockActions(w http.ResponseWriter, blob []byte) {
	var payload BlockActionPayload
	err := json.Unmarshal(blob, &payload)
	if err != nil {
		app.logger.Error("Could not parse BlockActionPayload", "error", err.Error())
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	if len(payload.Actions) == 0 {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	botToken, err := app.botToken(payload.Team.ID)
	if err != nil {
		app.logger.Error("Could not get bot token", "teamID", payload.Team.ID, "error", err.Error())
		http.Error(w, "Could not get bot token", http.StatusInternalServerError)
		return
	}

	actionID := payload.Actions[0].ActionID
	handler, ok := app.blockActions[actionID]
	if !ok {
		// Return 200 for unknown action IDs
		w.WriteHeader(http.StatusOK)
		return
	}

	ackChan := make(chan []byte)
	errChan := make(chan error)

	go func() {
		req := &BlockActionRequest{
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
		app.logger.Error("A block actions handler failed", "actionID", actionID, "error", err.Error())
		_, msgerr := req.Client.PostEphemeral(req.Payload.Channel.ID, req.Payload.User.ID, slack.MsgOptionText("An error occurred", false))
		if msgerr != nil {
			app.logger.Error("Unable to send error message to user", "user", req.Payload.User.ID, "error", msgerr.Error())
		}
		errChan <- err
	}()

	select {
	case <-ackChan:
		w.Write(nil)
	case err := <-errChan:
		if err == nil {
			return
		}
		http.Error(w, "An error occurred", http.StatusInternalServerError)
		return
	}
}
