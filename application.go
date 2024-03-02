// This package contains the Slap framework for developing Slack Applications.
package slap

import (
	"fmt"
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

// A function taking a Slack teamID (workspace ID) that returns
// the workspace's bot token as a string.
// For a non-distributed app, simply return your bot token.
type BotTokenGetter func(teamID string) (string, error)

// Configuration options for the Slap Application
type Config struct {
	// Required. A net/http Serve Mux.
	//
	// Slap will overwrite the following POST routes:
	// "POST {prefix}/interactions", "POST {prefix}/events", and "POST {prefix}/commands".
	Router *http.ServeMux
	// Optional. Adds a path to the start of the Slack routes.
	PathPrefix string
	// Required. Method for fetching bot tokens
	// for a workspace based on its team ID
	BotToken BotTokenGetter
	// Required. The Slack webhook signing secret for your app
	SigningSecret string
	// A logger for the Slap Application
	Logger *slog.Logger
}

// A Slap Application.
type Application struct {
	signingSecret   string
	botToken        BotTokenGetter
	commands        map[string]CommandHandler
	blockActions    map[string]BlockActionHandler
	viewSubmissions map[string]ViewSubmissionHandler
	events          map[string]EventHandler
	logger          *slog.Logger
}

// Registers a slash command handler.
//
// Panics if the slash command has already been registered.
func (app *Application) RegisterCommand(command string, handler CommandHandler) {
	_, ok := app.commands[command]
	if ok {
		panic(fmt.Sprintf("Command %v has already been registered", command))
	}
	app.commands[command] = handler
	app.logger.Info("Registered Command", "command", command)
}

// Registers a block action handler.
//
// Panics if the actionID has already been registered.
func (app *Application) RegisterBlockAction(actionID string, handler BlockActionHandler) {
	_, ok := app.blockActions[actionID]
	if ok {
		panic(fmt.Sprintf("Action ID %v has already been registered", actionID))
	}
	app.blockActions[actionID] = handler
	app.logger.Info("Registered Block Action", "actionID", actionID)
}

// Registers a view submission handler.
//
// Panics if the callbackID has already been registered.
func (app *Application) RegisterViewSubmission(callbackID string, handler ViewSubmissionHandler) {
	_, ok := app.viewSubmissions[callbackID]
	if ok {
		panic(fmt.Sprintf("View Callback ID %v has already been registered", callbackID))
	}
	app.viewSubmissions[callbackID] = handler
	app.logger.Info("Registered View Callback", "callbackID", callbackID)
}

// Registers an EventAPI event handler for a subscribed event type.
//
// Panics if the eventType has already been registered.
func (app *Application) RegisterEventHandler(eventType string, handler EventHandler) {
	_, ok := app.events[eventType]
	if ok {
		panic(fmt.Sprintf("Event Handler for type %v has already been registered", eventType))
	}
	app.events[eventType] = handler
	app.logger.Info("Registered Event Handler", "eventType", eventType)
}

// Creates a new Applcation with an http.ServeMux.
func New(config Config) *Application {
	if config.Router == nil {
		panic("Missing http.ServeMux in slap.New")
	}
	if config.SigningSecret == "" {
		panic("Missing Slack signing secret")
	}

	if config.BotToken == nil {
		panic("Missing Slack bot token getter")
	}

	logger := config.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{}))
	}

	app := Application{
		logger:          logger,
		botToken:        config.BotToken,
		signingSecret:   config.SigningSecret,
		commands:        make(map[string]CommandHandler),
		blockActions:    make(map[string]BlockActionHandler),
		viewSubmissions: make(map[string]ViewSubmissionHandler),
		events:          make(map[string]EventHandler),
	}

	config.Router.HandleFunc(fmt.Sprintf("POST %v/commands", config.PathPrefix), app.validateSignature(app.handleCommand))
	config.Router.HandleFunc(fmt.Sprintf("POST %v/interactions", config.PathPrefix), app.validateSignature(app.handleInteraction))
	config.Router.HandleFunc(fmt.Sprintf("POST %v/events", config.PathPrefix), app.validateSignature(app.handleEvent))

	return &app
}
