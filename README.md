# Slap: Easily build Slack Apps with Go

A Slack application framework inspired by [Slack's Bolt Framework](https://api.slack.com/bolt) and `net/http` library.

## Examples

### Slash Commands
```go
app.RegisterCommand("/hi", func(req *slap.CommandRequest) error {
    // Respond with 200 and an ephemeral message immediately
    req.AckWithAction(slap.CommandResponseAction{
        ResponseType: slap.RespondEphemeral,
        Text: "Hi, how are you?"
    })

    // Send another message!
    channel, ts, err := req.Client.PostEphemeral(req.Payload.ChannelID, req.Payload.UserID, slack.MsgOptionText("You said: " + req.Payload.Text))
    if err != nil {
        return err
    }

    // Open a modal!
    res, err := req.Client.OpenView(req.Payload.TriggerID, slack.ModalViewRequest{ 
        Type: "modal",
        CallbackID: "form-modal",
        Title: &slack.TextBlockObject{
            Type: "plain_text",
            Text: "Form"
        }
        ... 
    })

    return nil
})
```
### View Submissions
```go
app.RegisterViewSubmission("form-modal", func(req *slap.ViewSubmissionRequest) error {
    // Get a value from the view submission
    text := req.Payload.View.State.Values["text-input"]["text-input"]
    if !isTextValid(text) {
        // Respond with 200 and an error visible to the user
        req.AckWithAction(slap.ViewResponseAction{
            ResponseAction: slap.ViewResponseErrors,
            Errors: map[string]string{
                "text-input": "Please input a valid sentence"
            }
        })
        return nil
    }

    // Close the modal stack using the "clear" response action
    req.AckWithAction(slap.ViewResponseAction{
        ResponseAction: slap.ViewResponseClear
    })

    // Do something with the text value - save it to a store with the user's ID
    if err := store.save(req.Payload.User.ID, text); err != nil {
        return err
    }

    return nil
})

```

### Block Actions
```go
app.RegisterBlockAction("start-button", func(req *slap.BlockActionRequest) error {
    // Respond to Slack with 200
    req.Ack()

    // Open a modal!
    res, err := req.Client.OpenView(req.Payload.TriggerID, slack.ModalViewRequest{ 
        Type: "modal",
        CallbackID: "form-modal",
        Title: &slack.TextBlockObject{
            Type: "plain_text",
            Text: "Form"
        }
        ... 
    })

    return nil
})
```

### Events API
```go
app.RegisterEventHandler("message", func(req *slap.EventRequest) error {
    // Parse the inner Events API event
    var message slack.MessageEvent
    if err := json.Unmarshal(req.Payload.Event, &message); err != nil {
        return err
    }

    // Respond to Slack with 200
    req.Ack()

    // Ignore messages that your bot has sent or you will get stuck in recursive message hell
    if message.BotId != "" {
        return nil
    }

    // Do something with the message
    slog.Info("Received message", "message", message.Text)
    _, _, err := req.Client.PostMessage(message.Channel, slack.MsgOptionText("You wrote: " + message.Text, false))
    if err != nil {
        return err
    }

    return nil
})
```

## Quick Start
1. Create a Slack App at [api.slack.com/apps](https://api.slack.com/apps) and install it to your workspace (_Settings -> Install App_)
1. Set the following environment variables from your Slack App Settings
    ```
    export BOT_TOKEN={Settings -> Install App -> Bot User OAuth Token}
    ```
    ```
    export SIGNING_SECRET={Settings -> Basic Information -> Signing Secret}
    ```
1. Add the following to your `main.go`
    ```go
    package main

    import (
        "net/http"
        "os"

        "github.com/jacob-ian/slap"
    )

    func main() {
        router := http.NewServeMux()

        app := slap.New(slap.Config{
            Router: router,
            BotToken: func(teamID string) (string, error) {
                return os.Getenv("BOT_TOKEN"), nil
            },
            SigningSecret: os.Getenv("SIGNING_SECRET"),
        })

        app.RegisterCommand("/start", func(req *slap.CommandRequest) error {
            req.AckWithAction(slap.CommandResponseAction{
                ResponseType: slap.RespondInChannel,
                Text:         "Hello world!",
            })
            return nil
        })

        server := &http.Server{
            Addr:    ":4000",
            Handler: router,
        }
        panic(server.ListenAndServe())
    }
    ```
1. Run `go run main.go`
1. Use [ngrok](https://ngrok.com) to get a public URL for your local environment
1. Update your Slack App Settings:
    1. Slash Commands -> Create New Command 
        - Command: `/start`
        - Request URL: `https://{YOUR NGROK URL}/commands`
1. Use the `/start` command in your Slack Client

## Usage Guide
### Slack API Settings
To use Slap, you will need to update your Slack App's settings at [api.slack.com/apps](https://api.slack.com/apps).
#### Slash Commands
For all Slash Commands:
- Request URL: `https://{YOUR PUBLIC URL}/commands`
#### Interactivity & Shortcuts
- Turn on Interactivity
- Request URL: `https://{YOUR PUBLIC URL}/interactions`
#### Event Subscriptions
- Enable Events
- Request URL: `https://{YOUR PUBLIC URL}/events`
    - Slap will automatically complete URL verification
### Multiple Workspace Distribution
Slap supports app distribution to multiple workspaces with the `BotTokenGetter` in `slap.Config`:
```go
app := slap.New(slap.Config{
    ...,
    BotToken: func(teamID string) (string, error) {
        token, err := db.GetBotTokenByTeamID(teamID)
        if err != nil {
            return "", err
        }
        return token, nil
    }
})
```
This allows for the fetching of a workspace's bot token from your store by the workspace's Team ID, which is then used by the Slack API client.


## To Do
- [ ] Add shortcut support
- [ ] Add `view_closed` support
- [ ] Add `block_suggestion` support
- [ ] Add support for Gorilla Mux
- [ ] Add support for Echo

## Special Thanks

Thank you to the contributors at [github.com/go-slack/slack](https://github.com/go-slack/slack) for creating and maintaining the Slack API client and types
which are needed to make Slap work.
