package slap_test

import (
	"bytes"
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
	r := createSlackRequest("/interactions", []byte("Test"))

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

	body := "payload=\"" + string(payload) + "\""
	t.Log(body)

	w := httptest.NewRecorder()
	r := createSlackRequest("/interactions", []byte(body))
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
