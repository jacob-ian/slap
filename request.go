package slap

import (
	"log/slog"
	"net/http"

	"github.com/slack-go/slack"
)

type baseRequest struct {
	// A Slack API client
	Client *slack.Client
	// The logger as defined in Config
	Logger     *slog.Logger
	ackCalled  bool
	ackChannel chan []byte
	errChannel chan error
	writer     http.ResponseWriter
}

// Acknowledge Slack's request with Status 200
func (event *baseRequest) Ack() {
	if event.ackCalled {
		return
	}
	event.ackCalled = true
	event.ackChannel <- nil
}
