package slap

import (
	"encoding/json"
	"net/http"

	"github.com/slack-go/slack"
)

// The payload of the Slack view_submission request
type ViewSubmissionPayload struct {
	interactionPayload
	View slack.View `json:"view"`
}

// A Slack view submission request
type ViewSubmissionRequest struct {
	baseRequest
	Payload ViewSubmissionPayload
}

// A view response action ResponseType
type ViewResponseActionType string

// The ViewResponseAction ResponseType values
const (
	// Clears the modal stack
	ViewResponseClear ViewResponseActionType = "clear"
	// Displays an error to the user in the modal
	ViewResponseErrors ViewResponseActionType = "errors"
	// Updates the view of the currently open modal
	ViewResponseUpdate ViewResponseActionType = "update"
	// Adds a new modal to the modal stack
	ViewResponsePush ViewResponseActionType = "push"
)

// An immediate response action for a view submission
type ViewResponseAction struct {
	// The type of response: "clear", "errors", "update", or "push"
	ResponseAction ViewResponseActionType `json:"response_action"`
	// The view for "update" or "push" actions
	View *slack.View `json:"view,omitempty"`
	// The blockID-error map for "errors" actions.
	//
	// Example:
	//  errors := map[string]string {
	//    "email-input-block": "Please enter a valid email address."
	//  }
	Errors map[string]string `json:"errors,omitempty"`
}

// Immediately respond to Slack with a view response action
func (req *ViewSubmissionRequest) AckWithAction(action ViewResponseAction) {
	if req.ackCalled {
		return
	}
	req.ackCalled = true
	bytes, err := json.Marshal(action)
	println(string(bytes))
	if err != nil {
		req.Logger.Error("Could not encode view response action", "error", err.Error())
		req.errChannel <- err
		return
	}
	req.ackChannel <- bytes
}

// A function to handle a view submission request
type ViewSubmissionHandler func(req *ViewSubmissionRequest) error

func (app *Application) handleViewSubmission(w http.ResponseWriter, blob []byte) {
	var payload ViewSubmissionPayload
	err := json.Unmarshal(blob, &payload)
	if err != nil {
		app.logger.Error("Could not parse ViewSubmissionPayload", "error", err.Error())
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	botToken, err := app.botToken(payload.Team.ID)
	if err != nil {
		app.logger.Error("Could not get bot token", "teamID", payload.Team.ID, "error", err.Error())
		http.Error(w, "Could not get bot token", http.StatusInternalServerError)
		return
	}

	handler, ok := app.viewSubmissions[payload.View.CallbackID]
	if !ok {
		http.Error(w, "Invalid callback ID", http.StatusInternalServerError)
		return
	}

	ackChan := make(chan []byte)
	errChan := make(chan error)

	go func() {
		req := &ViewSubmissionRequest{
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
		app.logger.Error("A view submission handler failed", "callbackID", req.Payload.View.CallbackID, "error", err.Error())
		_, msgerr := req.Client.PostEphemeral(req.Payload.User.ID, req.Payload.User.ID, slack.MsgOptionText(app.errorMessage, false))
		if msgerr != nil {
			app.logger.Error("Unable to send error message to user", "user", req.Payload.User.ID, "error", msgerr.Error())
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
