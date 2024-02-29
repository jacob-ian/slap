package slap

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/slack-go/slack"
)

type baseEvent struct {
	// A Slack API client
	Client     *slack.Client
	ackCalled  bool
	ackChannel chan []byte
	errChannel chan error
	writer     http.ResponseWriter
}

// Acknowledge Slack's request with 200
func (event *baseEvent) Ack() {
	if event.ackCalled {
		return
	}
	event.ackCalled = true
	event.ackChannel <- nil
}

type BotTokenGetter func(teamID string) (string, error)

type Config struct {
	// Will overwrite the POST routes for "/interactions", "/events", and "/commands"
	Router *http.ServeMux
	// Adds a path to the start of the Slack routes
	PathPrefix string
	// Method for fetching bot tokens for a workspace based on team ID
	BotToken BotTokenGetter
	// The Slack webhook signing secret for your app
	SigningSecret string
}

type SlackApplication struct {
	botToken        BotTokenGetter
	commands        map[string]CommandHandler
	blockActions    map[string]BlockActionHandler
	viewSubmissions map[string]ViewSubmissionHandler
	events          map[string]EventHandler
}

// Register a slash command handler
func (app *SlackApplication) RegisterCommand(command string, handler CommandHandler) {
	_, ok := app.commands[command]
	if ok {
		panic(fmt.Sprintf("Command %v has already been registered", command))
	}
	app.commands[command] = handler
	slog.Info("Registered Command", "command", command)
}

// Register a block action handler
func (app *SlackApplication) RegisterBlockAction(actionID string, handler BlockActionHandler) {
	_, ok := app.blockActions[actionID]
	if ok {
		panic(fmt.Sprintf("Action ID %v has already been registered", actionID))
	}
	app.blockActions[actionID] = handler
	slog.Info("Registered Block Action", "actionID", actionID)
}

// Register a view submission handler
func (app *SlackApplication) RegisterViewSubmission(callbackID string, handler ViewSubmissionHandler) {
	_, ok := app.viewSubmissions[callbackID]
	if ok {
		panic(fmt.Sprintf("View Callback ID %v has already been registered", callbackID))
	}
	app.viewSubmissions[callbackID] = handler
	slog.Info("Registered View Callback", "callbackID", callbackID)
}

// Register an event handler for a subscribed event type
func (app *SlackApplication) RegisterEventHandler(eventType string, handler EventHandler) {
	_, ok := app.events[eventType]
	if ok {
		panic(fmt.Sprintf("Event Handler for type %v has already been registered", eventType))
	}
	app.events[eventType] = handler
	slog.Info("Registered Event Handler", "eventType", eventType)
}

func New(config Config) *SlackApplication {
	app := SlackApplication{
		botToken:        config.BotToken,
		commands:        make(map[string]CommandHandler),
		blockActions:    make(map[string]BlockActionHandler),
		viewSubmissions: make(map[string]ViewSubmissionHandler),
		events:          make(map[string]EventHandler),
	}

	config.Router.HandleFunc(fmt.Sprintf("POST %v/commands", config.PathPrefix), verify(config.SigningSecret, app.handleCommand))
	config.Router.HandleFunc(fmt.Sprintf("POST %v/interactions", config.PathPrefix), verify(config.SigningSecret, app.handleInteraction))
	config.Router.HandleFunc(fmt.Sprintf("POST %v/events", config.PathPrefix), verify(config.SigningSecret, app.handleEvent))

	return &app
}
