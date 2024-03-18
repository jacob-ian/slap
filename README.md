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

## To Do
- [ ] Support shortcuts

## Special Thanks
Thank you to the contributors at [github.com/go-slack/slack](https://github.com/go-slack/slack) for creating and maintaining the Slack API client and types
which are needed to make Slap work.
