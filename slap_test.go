package slap_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/jacob-ian/slap"
)

func createTestApp() (*slap.Application, *http.ServeMux) {
	router := http.NewServeMux()
	return slap.New(slap.Config{
		Router: router,
		BotToken: func(teamID string) (string, error) {
			return "test", nil
		},
		SigningSecret: "signing-secret",
	}), router
}

func createSlackRequest(path string, body []byte) *http.Request {
	r := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))

	ts := fmt.Sprintf("%v", time.Now().UnixMilli())
	contents := "v0:" + ts + ":" + string(body)
	hmac := hmac.New(sha256.New, []byte("signing-secret"))
	_, err := hmac.Write([]byte(contents))
	if err != nil {
		panic("Could not write hmac:" + err.Error())
	}

	signature := "v0=" + hex.EncodeToString(hmac.Sum(nil))
	r.Header.Add("x-slack-request-timestamp", ts)
	r.Header.Add("x-slack-signature", signature)

	return r
}

func getJSONTestData(name string) ([]byte, error) {
	b, err := os.ReadFile(fmt.Sprintf("testdata/%v", name))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = json.Compact(&buf, b)
	if err != nil {
		return nil, err
	}
	out, err := io.ReadAll(&buf)
	if err != nil {
		return nil, err
	}
	return out, nil
}
