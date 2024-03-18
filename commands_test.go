package slap_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/jacob-ian/slap"
)

func testCommandBody() []byte {
	payload := make(url.Values)
	payload.Add("command", "/help")
	payload.Add("text", "me")
	payload.Add("team_id", "T0123456")
	payload.Add("team_domain", "slap")
	payload.Add("channel_id", "C0123456")
	payload.Add("channel_name", "slap-test")
	payload.Add("user_id", "U0123456")
	payload.Add("user_name", "jacob-ian")
	payload.Add("response_url", "https://slack.com/api")
	payload.Add("trigger_id", "abcd1234")
	payload.Add("api_app_id", "A0123456")
	return []byte(payload.Encode())
}

func TestCommandNoSignatureHeader(t *testing.T) {
	t.Parallel()

	app, router := createTestApp()
	app.RegisterCommand("/help", func(req *slap.CommandRequest) error {
		req.Ack()
		return nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/commands", bytes.NewReader([]byte("Hello")))
	r.Header.Add("content-type", "application/x-www-form-urlencoded")

	router.ServeHTTP(w, r)
	res := w.Result()

	statusGot, statusWant := res.StatusCode, http.StatusUnauthorized
	if statusGot != statusWant {
		t.Errorf("Unexpected status code, got: %v, want: %v", statusGot, statusWant)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("Could not ready body: %v", err.Error())
	}

	textGot, textWant := string(body), "Unauthenticated\n"
	if textGot != textWant {
		t.Errorf("Unexpected body text, got: %v, want: %v", textGot, textWant)
	}
}

func TestCommandBadPayload(t *testing.T) {
	t.Parallel()

	app, router := createTestApp()
	app.RegisterCommand("/help", func(req *slap.CommandRequest) error {
		req.Ack()
		return nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/commands", bytes.NewReader([]byte("Hello")))
	addSignatureHeaders(r)

	router.ServeHTTP(w, r)
	res := w.Result()

	statusGot, statusWant := res.StatusCode, http.StatusBadRequest
	if statusGot != statusWant {
		t.Errorf("Unexpected status code, got: %v, want: %v", statusGot, statusWant)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("Could not ready body: %v", err.Error())
	}

	textGot, textWant := string(body), "Bad Request\n"
	if textGot != textWant {
		t.Errorf("Unexpected body text, got: %v, want: %v", textGot, textWant)
	}
}

func TestCommandNoHandler(t *testing.T) {
	t.Parallel()

	_, router := createTestApp()

	payload := testCommandBody()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/commands", bytes.NewReader(payload))
	r.Header.Add("content-type", "application/x-www-form-urlencoded")
	addSignatureHeaders(r)

	router.ServeHTTP(w, r)
	res := w.Result()

	statusGot, statusWant := res.StatusCode, http.StatusBadRequest
	if statusGot != statusWant {
		t.Errorf("Unexpected status code, got: %v, want: %v", statusGot, statusWant)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("Could not ready body: %v", err.Error())
	}

	textGot, textWant := string(body), "Invalid command\n"
	if textGot != textWant {
		t.Errorf("Unexpected body text, got: %v, want: %v", textGot, textWant)
	}
}

func TestCommandHandlerAck(t *testing.T) {
	t.Parallel()

	app, router := createTestApp()
	app.RegisterCommand("/help", func(req *slap.CommandRequest) error {
		req.Ack()
		return nil
	})

	payload := testCommandBody()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/commands", bytes.NewReader(payload))
	r.Header.Add("content-type", "application/x-www-form-urlencoded")
	addSignatureHeaders(r)

	router.ServeHTTP(w, r)
	res := w.Result()

	statusGot, statusWant := res.StatusCode, http.StatusOK
	if statusGot != statusWant {
		t.Errorf("Unexpected status code, got: %v, want: %v", statusGot, statusWant)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("Could not ready body: %v", err.Error())
	}

	textGot, textWant := string(body), ""
	if textGot != textWant {
		t.Errorf("Unexpected body text, got: %v, want: %v", textGot, textWant)
	}
}

func TestCommandHandlerError(t *testing.T) {
	t.Parallel()

	app, router := createTestApp()
	app.RegisterCommand("/help", func(req *slap.CommandRequest) error {
		return errors.New("Error")
	})

	payload := testCommandBody()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/commands", bytes.NewReader(payload))
	r.Header.Add("content-type", "application/x-www-form-urlencoded")
	addSignatureHeaders(r)

	router.ServeHTTP(w, r)
	res := w.Result()

	statusGot, statusWant := res.StatusCode, http.StatusInternalServerError
	if statusGot != statusWant {
		t.Errorf("Unexpected status code, got: %v, want: %v", statusGot, statusWant)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("Could not ready body: %v", err.Error())
	}

	textGot, textWant := string(body), "An error occurred\n"
	if textGot != textWant {
		t.Errorf("Unexpected body text, got: %v, want: %v", textGot, textWant)
	}
}

func TestCommandHandlerAction(t *testing.T) {
	t.Parallel()

	app, router := createTestApp()
	app.RegisterCommand("/help", func(req *slap.CommandRequest) error {
		req.AckWithAction(slap.CommandResponseAction{
			ResponseType: slap.RespondInChannel,
			Text:         "Howdy!",
		})
		return nil
	})

	payload := testCommandBody()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/commands", bytes.NewReader(payload))
	r.Header.Add("content-type", "application/x-www-form-urlencoded")
	addSignatureHeaders(r)

	router.ServeHTTP(w, r)
	res := w.Result()

	statusGot, statusWant := res.StatusCode, http.StatusOK
	if statusGot != statusWant {
		t.Errorf("Unexpected status code, got: %v, want: %v", statusGot, statusWant)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("Could not ready body: %v", err.Error())
	}

	textGot, textWant := string(body), `{"response_type":"in_channel","text":"Howdy!"}`
	if textGot != textWant {
		t.Errorf("Unexpected body text, got: %v, want: %v", textGot, textWant)
	}
}
