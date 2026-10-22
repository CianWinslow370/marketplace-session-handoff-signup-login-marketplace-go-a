package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestVerifyCaptchaIncludesWidgetRecordID(t *testing.T) {
	t.Setenv("INFRAI_API_KEY", "test-key")
	oldClient := http.DefaultClient
	defer func() { http.DefaultClient = oldClient }()
	http.DefaultClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["widget_record_id"] != "widget-1" || body["token"] != "captcha-token" {
			t.Fatalf("captcha request: %v", body)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Header: make(http.Header)}, nil
	})}
	if err := verifyCaptcha(httptest.NewRequest("POST", "/signup", nil), "widget-1", "captcha-token"); err != nil {
		t.Fatal(err)
	}
}

func TestPaidOrderHandoff(t *testing.T) {
	s := newStore()
	req := httptest.NewRequest("POST", "/orders/ord-1/handoff", strings.NewReader(""))
	rec := httptest.NewRecorder()
	s.handoff(rec, req)
	if rec.Code != 200 || s.orders["ord-1"].Status != "handed_off" {
		t.Fatalf("handoff decision: code=%d status=%s", rec.Code, s.orders["ord-1"].Status)
	}
	bad := httptest.NewRequest("POST", "/orders/missing/handoff", nil)
	out := httptest.NewRecorder()
	s.handoff(out, bad)
	if out.Code != 409 {
		t.Fatalf("missing order code=%d", out.Code)
	}
}
