package slap

import (
	"encoding/json"
	"net/http"
)

type interactionPayloadType struct {
	Type string `json:"type"`
}

type interactionPayload struct {
	interactionPayloadType
	Team struct {
		ID     string `json:"id"`
		Domain string `json:"domain"`
	} `json:"team"`
	User struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		TeamID   string `json:"team_id"`
	} `json:"user"`
	TriggerID string `json:"trigger_id"`
	ApiAppId  string `json:"api_app_id"`
}

func (app *Application) handleInteraction(w http.ResponseWriter, r *http.Request) {
	blob := []byte(r.FormValue("payload"))

	var payloadType interactionPayloadType
	err := json.Unmarshal(blob, &payloadType)
	if err != nil {
		app.logger.Error("Could not parse payload interactions type", "error", err.Error())
		http.Error(w, "Bad Payload", http.StatusBadRequest)
		return
	}

	if payloadType.Type == "view_submission" {
		app.handleViewSubmission(w, blob)
		return
	} else if payloadType.Type == "block_actions" {
		app.handleBlockActions(w, blob)
		return
	} else {
		http.Error(w, "Unknown interaction type", http.StatusInternalServerError)
		return
	}
}
