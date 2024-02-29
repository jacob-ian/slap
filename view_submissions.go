package slap

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/slack-go/slack"
)

type ViewSubmissionPayload struct {
	interactionPayload
	View slack.View `json:"view"`
}

type ViewSubmissionRequest struct {
	baseEvent
	Payload ViewSubmissionPayload
}

func (event *ViewSubmissionRequest) AckWithAction(action slack.ViewSubmissionResponse) {
	if event.ackCalled {
		return
	}
	event.ackCalled = true
	bytes, err := json.Marshal(action)
	if err != nil {
		slog.Error("Could not encode view response action", "error", err.Error())
		event.errChannel <- err
		return
	}
	event.ackChannel <- bytes
}

type ViewSubmissionHandler func(req *ViewSubmissionRequest) error

func (app *SlackApplication) handleViewSubmission(w http.ResponseWriter, blob []byte) {
	var payload ViewSubmissionPayload
	err := json.Unmarshal(blob, &payload)
	if err != nil {
		slog.Warn("Could not parse ViewSubmissionPayload", "error", err.Error())
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	botToken, err := app.botToken(payload.Team.ID)
	if err != nil {
		slog.Error("Could not find bot token", "teamID", payload.Team.ID, "error", err.Error())
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
		slog.Error("A view submission handler failed", "callbackID", req.Payload.View.CallbackID, "error", err.Error())
		_, msgerr := req.Client.PostEphemeral(req.Payload.User.ID, req.Payload.User.ID, slack.MsgOptionText("An error occurred", false))
		if msgerr != nil {
			slog.Error("Unable to send error message to user", "user", req.Payload.User.ID, "error", msgerr.Error())
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
