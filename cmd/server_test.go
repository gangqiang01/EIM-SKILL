package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServerGetConfigReturnsFullConfigObject(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","token":"test-token"}`))
		case "/api/configmgr/getqrcodedata":
			if got := r.Header.Get("accesstoken"); got != "test-token" {
				t.Fatalf("accesstoken header = %q, want %q", got, "test-token")
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"WP.SRP","data":"ciphertext"}`))
		default:
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}
	}))
	defer ts.Close()

	origBaseURL := baseURL
	origUsername := username
	origPassword := password
	t.Cleanup(func() {
		baseURL = origBaseURL
		username = origUsername
		password = origPassword
	})

	baseURL = ts.URL
	username = "tester"
	password = "secret"

	text := withCapturedOutput(func() {
		if err := serverGetConfigCmd.RunE(serverGetConfigCmd, nil); err != nil {
			printErrorResult(err)
		}
	})

	var got struct {
		Status int                    `json:"status"`
		Msg    string                 `json:"msg"`
		Data   map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("unmarshal command output: %v\noutput: %s", err, text)
	}

	if got.Status != statusOK {
		t.Fatalf("status = %d, want %d; output: %s", got.Status, statusOK, text)
	}
	if got.Msg != "" {
		t.Fatalf("msg = %q, want empty", got.Msg)
	}
	if got.Data["id"] != "WP.SRP" {
		t.Fatalf("data.id = %#v, want %q", got.Data["id"], "WP.SRP")
	}
	if got.Data["data"] != "ciphertext" {
		t.Fatalf("data.data = %#v, want %q", got.Data["data"], "ciphertext")
	}
	if len(got.Data) != 2 {
		t.Fatalf("data field count = %d, want 2", len(got.Data))
	}
}