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

// Acknowledge Slack's request with 200
func (event *baseRequest) Ack() {
	if event.ackCalled {
		return
	}
	event.ackCalled = true
	event.ackChannel <- nil
}

type BotTokenGetter func(teamID string) (string, error)

// Configuration options for the SlackApplication
type Config struct {
	// A net/http Serve Mux. Slap will overwrite the POST routes for "/interactions", "/events", and "/commands"
	Router *http.ServeMux
	// Adds a path to the start of the Slack routes
	PathPrefix string
	// Method for fetching bot tokens for a workspace based on team ID
	BotToken BotTokenGetter
	// The Slack webhook signing secret for your app
	SigningSecret string
	// A logger
	Logger *slog.Logger
}

type SlackApplication struct {
	signingSecret   string
	botToken        BotTokenGetter
	commands        map[string]CommandHandler
	blockActions    map[string]BlockActionHandler
	viewSubmissions map[string]ViewSubmissionHandler
	events          map[string]EventHandler
	logger          *slog.Logger
}

// Register a slash command handler
func (app *SlackApplication) RegisterCommand(command string, handler CommandHandler) {
	_, ok := app.commands[command]
	if ok {
		panic(fmt.Sprintf("Command %v has already been registered", command))
	}
	app.commands[command] = handler
	app.logger.Info("Registered Command", "command", command)
}

// Register a block action handler
// Panics if the actionID has already been registered
func (app *SlackApplication) RegisterBlockAction(actionID string, handler BlockActionHandler) {
	_, ok := app.blockActions[actionID]
	if ok {
		panic(fmt.Sprintf("Action ID %v has already been registered", actionID))
	}
	app.blockActions[actionID] = handler
	app.logger.Info("Registered Block Action", "actionID", actionID)
}

// Register a view submission handler
// Panics if the callbackID has already been registered
func (app *SlackApplication) RegisterViewSubmission(callbackID string, handler ViewSubmissionHandler) {
	_, ok := app.viewSubmissions[callbackID]
	if ok {
		panic(fmt.Sprintf("View Callback ID %v has already been registered", callbackID))
	}
	app.viewSubmissions[callbackID] = handler
	app.logger.Info("Registered View Callback", "callbackID", callbackID)
}

// Register an event handler for a subscribed event type
// Panics if the eventType has already been registered
func (app *SlackApplication) RegisterEventHandler(eventType string, handler EventHandler) {
	_, ok := app.events[eventType]
	if ok {
		panic(fmt.Sprintf("Event Handler for type %v has already been registered", eventType))
	}
	app.events[eventType] = handler
	app.logger.Info("Registered Event Handler", "eventType", eventType)
}

// Creates a new Slack Application with an http.ServeMux
func New(config Config) *SlackApplication {
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

	app := SlackApplication{
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
