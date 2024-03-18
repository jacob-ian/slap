package slap_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jacob-ian/slap"
)

func TestBlockActionsNoSignatureHeader(t *testing.T) {
	t.Parallel()

	app, router := createTestApp()
	app.RegisterBlockAction("test-action", func(req *slap.BlockActionRequest) error {
		req.Ack()
		return nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/interactions", bytes.NewReader([]byte("Hello")))
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

func TestBlockActionsBadPayload(t *testing.T) {
	t.Parallel()

	app, router := createTestApp()
	app.RegisterBlockAction("test-action", func(req *slap.BlockActionRequest) error {
		req.Ack()
		return nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/interactions", bytes.NewReader([]byte("Test")))
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

	textGot, textWant := string(body), "Bad Payload\n"
	if textGot != textWant {
		t.Errorf("Unexpected body text, got: %v, want: %v", textGot, textWant)
	}
}

func TestBlockActionsNoAction(t *testing.T) {
	t.Parallel()

	app, router := createTestApp()
	app.RegisterBlockAction("test-action", func(req *slap.BlockActionRequest) error {
		req.Ack()
		return nil
	})

	payload, err := getJSONTestData("block_actions_empty.json")
	if err != nil {
		t.Errorf("Could not get testdata: %v", err.Error())
	}

	body := "payload=" + string(payload)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/interactions", bytes.NewReader([]byte(body)))
	r.Header.Add("content-type", "application/x-www-form-urlencoded")
	addSignatureHeaders(r)

	router.ServeHTTP(w, r)
	res := w.Result()

	statusGot, statusWant := res.StatusCode, http.StatusBadRequest
	if statusGot != statusWant {
		t.Errorf("Unexpected status code, got: %v, want: %v", statusGot, statusWant)
	}

	bodyGot, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("Could not read body: %v", err.Error())
	}

	textGot, textWant := string(bodyGot), "Invalid payload\n"
	if textGot != textWant {
		t.Errorf("Unexpected body text, got: %v, want: %v", textGot, textWant)
	}
}

func TestBlockActionsNoHandler(t *testing.T) {
	t.Parallel()

	_, router := createTestApp()

	payload, err := getJSONTestData("block_actions_msg_button.json")
	if err != nil {
		t.Errorf("Could not get testdata: %v", err.Error())
	}

	body := "payload=" + string(payload)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/interactions", bytes.NewReader([]byte(body)))
	r.Header.Add("content-type", "application/x-www-form-urlencoded")
	addSignatureHeaders(r)

	router.ServeHTTP(w, r)
	res := w.Result()

	statusGot, statusWant := res.StatusCode, http.StatusOK
	if statusGot != statusWant {
		t.Errorf("Unexpected status code, got: %v, want: %v", statusGot, statusWant)
	}

	bodyGot, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("Could not read res body: %v", err.Error())
	}

	if string(bodyGot) != "" {
		t.Errorf("Unexpected body, got: %v, want: empty string", string(bodyGot))
	}
}

func TestBlockActionsAck(t *testing.T) {
	t.Parallel()

	app, router := createTestApp()
	app.RegisterBlockAction("test-action", func(req *slap.BlockActionRequest) error {
		req.Ack()
		return nil
	})

	payload, err := getJSONTestData("block_actions_msg_button.json")
	if err != nil {
		t.Errorf("Could not get testdata: %v", err.Error())
	}

	body := "payload=" + string(payload)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/interactions", bytes.NewReader([]byte(body)))
	r.Header.Add("content-type", "application/x-www-form-urlencoded")
	addSignatureHeaders(r)

	router.ServeHTTP(w, r)
	res := w.Result()

	statusGot, statusWant := res.StatusCode, http.StatusOK
	if statusGot != statusWant {
		t.Errorf("Unexpected status code, got: %v, want: %v", statusGot, statusWant)
	}

	bodyGot, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("Could not read res body: %v", err.Error())
	}

	if string(bodyGot) != "" {
		t.Errorf("Unexpected body, got: %v, want: empty string", string(bodyGot))
	}
}

func TestBlockActionsHandlerError(t *testing.T) {
	t.Parallel()

	app, router := createTestApp()
	app.RegisterBlockAction("test-action", func(req *slap.BlockActionRequest) error {
		return errors.New("Error")
	})

	payload, err := getJSONTestData("block_actions_msg_button.json")
	if err != nil {
		t.Errorf("Could not get testdata: %v", err.Error())
	}

	body := "payload=" + string(payload)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/interactions", bytes.NewReader([]byte(body)))
	r.Header.Add("content-type", "application/x-www-form-urlencoded")
	addSignatureHeaders(r)

	router.ServeHTTP(w, r)
	res := w.Result()

	statusGot, statusWant := res.StatusCode, http.StatusInternalServerError
	if statusGot != statusWant {
		t.Errorf("Unexpected status code, got: %v, want: %v", statusGot, statusWant)
	}

	bodyGot, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("Could not read res body: %v", err.Error())
	}

	textGot, textWant := string(bodyGot), "An error occurred\n"
	if textGot != textWant {
		t.Errorf("Unexpected body, got: %v, want: %v", textGot, textWant)
	}
}
