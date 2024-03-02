package slap

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/slack-go/slack"
)

type OuterEventType string

const (
	EventCallback   OuterEventType = "event_callback"
	UrlVerification OuterEventType = "url_verification"
	RateLimited     OuterEventType = "app_rate_limited"
)

type baseOuterEvent struct {
	Type           OuterEventType `json:"type"`
	TeamID         string         `json:"team_id"`
	ApiAppId       string         `json:"api_app_id"`
	Authorizations []struct {
		EnterpriseID string `json:"enterprise_id"`
		TeamID       string `json:"team_id"`
		UserID       string `json:"user_id"`
		IsBot        bool   `json:"is_bot"`
	} `json:"authorizations"`
	EventContext string `json:"event_context"`
	EventID      string `json:"event_id"`
	EventTime    uint64 `json:"event_time"`
}

type urlVerificationEvent struct {
	Challenge string `json:"challenge"`
}

type appRateLimitedEvent struct {
	MinuteRateLimited uint64 `json:"minute_rate_limited"`
}

type outerEvent struct {
	baseOuterEvent
	urlVerificationEvent
	appRateLimitedEvent
	Event json.RawMessage `json:"event"`
}

type innerEventType struct {
	Type string `json:"type"`
}

type baseInnerEvent struct {
	innerEventType
	User           string `json:"user"`
	EventTimestamp string `json:"event_ts"`
	Timestamp      string `json:"ts"`
}

type EventPayload struct {
	baseOuterEvent
	Event json.RawMessage
}

type MessageEvent struct {
	baseInnerEvent
	slack.MessageEvent
}

func (e *MessageEvent) IsBot() bool {
	return e.BotID != ""
}

type EventRequest struct {
	baseRequest
	Payload EventPayload
}

type EventHandler func(req *EventRequest) error

func (app *SlackApplication) handleEvent(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		app.logger.Error("Could not read event request body", "error", err.Error())
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var outer outerEvent
	err = json.Unmarshal(body, &outer)
	if err != nil {
		app.logger.Error("Invalid outer event payload", "error", err.Error())
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	switch outer.Type {
	case UrlVerification:
		w.Header().Add("content-type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(outer.Challenge))
	case RateLimited:
		w.WriteHeader(http.StatusOK)
		app.logger.Warn("Events API has been rate limited", "minute_limited", outer.MinuteRateLimited)
	case EventCallback:
		app.handleEventCallback(w, outer)
	default:
		app.logger.Warn("Unknown outer event type", "type", outer.Type)
		http.Error(w, "Unknown outer event type", http.StatusBadRequest)
	}
}

func (app *SlackApplication) handleEventCallback(w http.ResponseWriter, o outerEvent) {
	var innerType innerEventType
	err := json.Unmarshal(o.Event, &innerType)
	if err != nil {
		app.logger.Error("Could not parse inner event type", "error", err.Error())
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	handler, ok := app.events[innerType.Type]
	if !ok {
		// Return 200 if event types without handlers are received
		app.logger.Warn("No handler registered for event", "eventType", innerType.Type)
		w.WriteHeader(http.StatusOK)
		return
	}

	botToken, err := app.botToken(o.TeamID)
	if err != nil {
		app.logger.Error("Could not get bot token", "teamID", o.TeamID, "error", err.Error())
		http.Error(w, "An error occurred", http.StatusInternalServerError)
		return
	}

	ackChan := make(chan []byte)
	errChan := make(chan error)

	go func() {
		req := &EventRequest{
			baseRequest: baseRequest{
				Client:     slack.New(botToken),
				Logger:     app.logger,
				writer:     w,
				ackCalled:  false,
				ackChannel: ackChan,
				errChannel: errChan,
			},
			Payload: EventPayload{
				baseOuterEvent: o.baseOuterEvent,
				Event:          o.Event,
			},
		}
		err := handler(req)
		if err == nil {
			return
		}
		app.logger.Error("An event handler failed", "eventType", innerType.Type, "error", err.Error())
		errChan <- err
	}()

	select {
	case ack := <-ackChan:
		w.Write(ack)
	case err := <-errChan:
		if err == nil {
			return
		}
		http.Error(w, "An error occurred", http.StatusInternalServerError)
		return
	}
}
