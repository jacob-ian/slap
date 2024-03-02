package slap

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"io"
	"net/http"
)

func (app *SlackApplication) validateSignature(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		signature := r.Header.Get("x-slack-signature")
		timestamp := r.Header.Get("x-slack-request-timestamp")
		if signature == "" || timestamp == "" {
			http.Error(w, "Unauthenticated", http.StatusUnauthorized)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			app.logger.Error("Could not read request body", "error", err.Error())
			http.Error(w, "Internal Error", http.StatusInternalServerError)
			return
		}

		r.Body = io.NopCloser(bytes.NewBuffer(body))

		contents := "v0:" + timestamp + ":" + string(body)
		hmac := hmac.New(sha256.New, []byte(app.signingSecret))
		_, err = hmac.Write([]byte(contents))
		if err != nil {
			app.logger.Error("Could not write HMAC", "error", err.Error())
			http.Error(w, "Internal Error", http.StatusInternalServerError)
			return
		}

		hmacHex := hex.EncodeToString(hmac.Sum(nil))
		expected := "v0=" + hmacHex

		if subtle.ConstantTimeCompare([]byte(expected), []byte(signature)) == 0 {
			http.Error(w, "Unauthenticated", http.StatusUnauthorized)
			return
		}

		handler(w, r)
	}
}
