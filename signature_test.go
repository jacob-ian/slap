package slap

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func createApp() *Application {
	return New(Config{
		Router: http.NewServeMux(),
		BotToken: func(teamID string) (string, error) {
			return "test", nil
		},
		SigningSecret: "secret",
	})
}

func TestValidateSignature(t *testing.T) {
	t.Run("Should return 401 if missing headers", func(t *testing.T) {
		t.Parallel()
		h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Write([]byte{})
		})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/interactions", bytes.NewReader([]byte("Hello")))
		createApp().validateSignature(h)(w, r)
		res := w.Result()

		if res.StatusCode != http.StatusUnauthorized {
			t.Errorf("Status code is invalid, got: %v, want: %v", res.StatusCode, http.StatusUnauthorized)
		}

		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Errorf("Could not read body: %v", err.Error())
		}

		text := string(body)
		if text != "Unauthenticated\n" {
			t.Errorf("Response text is invalid, got: %v, want: %v", text, "Unauthenticated")
		}
	})

	t.Run("Should return 401 if secret is invalid", func(t *testing.T) {
		t.Parallel()

		h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Write([]byte{})
		})
		w := httptest.NewRecorder()
		body := "Hello"
		r := httptest.NewRequest(http.MethodPost, "/interactions", bytes.NewReader([]byte(body)))

		ts := "1710311551993"
		contents := "v0:" + ts + ":" + body

		hmac := hmac.New(sha256.New, []byte("bad-secret"))
		_, err := hmac.Write([]byte(contents))
		if err != nil {
			t.Errorf("Could not write hmac: %v", err.Error())
		}

		signature := "v0=" + hex.EncodeToString(hmac.Sum(nil))

		r.Header.Add("x-slack-request-timestamp", ts)
		r.Header.Add("x-slack-signature", signature)

		createApp().validateSignature(h)(w, r)
		res := w.Result()

		if res.StatusCode != http.StatusUnauthorized {
			t.Errorf("Unexpected status code, got: %v, want: %v", res.StatusCode, http.StatusUnauthorized)
		}

		bodyOut, err := io.ReadAll(res.Body)
		if err != nil {
			t.Errorf("Could not read body: %v", err.Error())
		}

		text := string(bodyOut)
		if text != "Unauthenticated\n" {
			t.Errorf("Response text is invalid, got: %v, want: %v", text, "Unauthenticated")
		}
	})
}
