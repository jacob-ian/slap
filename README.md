# Slap!

A framework for building Slack Applications in Go!

# Example
## Slash Commands
```go
app.RegisterCommand("/hi", func(req *slap.CommandRequest) error) {
    // Respond with an ephemeral message straight away
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
        ... 
    })

    return nil
}
```

# Special Thanks
Thank you to the contributors at (github.com/go-slack/slack)[https://github.com/go-slack/slack] for creating and maintaining the Slack API client and types
which are needed to make Slap work.
