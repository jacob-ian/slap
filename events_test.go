package slap_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jacob-ian/slap"
	"github.com/slack-go/slack"
)

func TestEventsNoSignatureHeader(t *testing.T) {
	t.Parallel()

	app, router := createTestApp()
	app.RegisterEventHandler("message", func(req *slap.EventRequest) error {
		req.Ack()
		return nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader([]byte("Hello")))
	r.Header.Add("content-type", "application/json")

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

func TestEventsRateLimit(t *testing.T) {
	t.Parallel()

	app, router := createTestApp()
	app.RegisterEventHandler("message", func(req *slap.EventRequest) error {
		req.Ack()
		return nil
	})

	payload, err := getJSONTestData("event_rate_limited.json")
	if err != nil {
		t.Errorf("Could not get testdata: %v", err.Error())
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(payload))
	r.Header.Add("content-type", "application/json")
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

func TestEventsUrlVerification(t *testing.T) {
	t.Parallel()

	app, router := createTestApp()
	app.RegisterEventHandler("message", func(req *slap.EventRequest) error {
		req.Ack()
		return nil
	})

	payload, err := getJSONTestData("event_url_verification.json")
	if err != nil {
		t.Errorf("Could not get testdata: %v", err.Error())
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(payload))
	r.Header.Add("content-type", "application/json")
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

	textGot, textWant := string(body), "ea0bb9129a4ab50da8714fc116b70a0d"
	if textGot != textWant {
		t.Errorf("Unexpected body text, got: %v, want: %v", textGot, textWant)
	}
}

func TestEventCallbackNoHandler(t *testing.T) {
	t.Parallel()

	app, router := createTestApp()
	app.RegisterEventHandler("app_home_opened", func(req *slap.EventRequest) error {
		req.Ack()
		return nil
	})

	payload, err := getJSONTestData("event_message.json")
	if err != nil {
		t.Errorf("Could not get testdata: %v", err.Error())
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(payload))
	r.Header.Add("content-type", "application/json")
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

func TestEventCallbackHandlerAck(t *testing.T) {
	t.Parallel()

	app, router := createTestApp()
	app.RegisterEventHandler("message", func(req *slap.EventRequest) error {
		req.Ack()
		return nil
	})

	payload, err := getJSONTestData("event_message.json")
	if err != nil {
		t.Errorf("Could not get testdata: %v", err.Error())
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(payload))
	r.Header.Add("content-type", "application/json")
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

func TestEventCallbackHandlerError(t *testing.T) {
	t.Parallel()

	app, router := createTestApp()
	app.RegisterEventHandler("message", func(req *slap.EventRequest) error {
		return errors.New("Error")
	})

	payload, err := getJSONTestData("event_message.json")
	if err != nil {
		t.Errorf("Could not get testdata: %v", err.Error())
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(payload))
	r.Header.Add("content-type", "application/json")
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

func TestEventCallbackParseMessage(t *testing.T) {
	t.Parallel()

	app, router := createTestApp()
	app.RegisterEventHandler("message", func(req *slap.EventRequest) error {
		var message slack.MessageEvent
		err := json.Unmarshal(req.Payload.Event, &message)
		if err != nil {
			return err
		}
		if message.Text != "Hello world" {
			return errors.New("Bad parse")
		}
		req.Ack()
		return nil
	})

	payload, err := getJSONTestData("event_message.json")
	if err != nil {
		t.Errorf("Could not get testdata: %v", err.Error())
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(payload))
	r.Header.Add("content-type", "application/json")
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
