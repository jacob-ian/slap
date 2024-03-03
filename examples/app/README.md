# Example: app

This example shows the registration of a slash command "/start",
a block action (from a button press) "start-button", a view submission 
callback "form-modal", and an event handler for "message". 

To run this example:
1. Set your App's signing secret and bot token in your environment
2. Run `go run app.go`
3. Start `ngrok` with port 4000 
4. Update your Slack app in [api.slack.com/apps](https://api.slack.com/apps) with:
    1. Commands: Add `/start` with the URL `{ngrok_url}/commands`
    1. Interactions: Use `{ngrok_url}/interactions` as the endpoint
    1. Events: Subscribe to the "message" event type and use `{ngrok_url}/events` for the Events API endpoint
5. Run the `/start` command in Slack
