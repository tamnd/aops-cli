package aops

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetUserAgent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		fmt.Fprint(w, "ok")
	}))
	defer ts.Close()

	cfg := DefaultConfig()
	cfg.Rate = 0
	cfg.Retries = 0
	c := NewClient(cfg)
	body, err := c.get(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if string(body) != "ok" {
		t.Errorf("body = %q, want ok", body)
	}
}

func TestGetRetryOn503(t *testing.T) {
	var hits int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		fmt.Fprint(w, "recovered")
	}))
	defer ts.Close()

	cfg := DefaultConfig()
	cfg.Rate = 0
	cfg.Retries = 5
	cfg.Timeout = 0
	c := NewClient(cfg)
	// Patch retryWait to be instant for tests.
	body, err := c.get(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("get after retries: %v", err)
	}
	if string(body) != "recovered" {
		t.Errorf("body = %q after retries", body)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
}

func TestGetBlockedOn404(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cfg := DefaultConfig()
	cfg.Rate = 0
	cfg.Retries = 0
	c := NewClient(cfg)
	_, err := c.get(context.Background(), ts.URL)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetBlockedOn403(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer ts.Close()

	cfg := DefaultConfig()
	cfg.Rate = 0
	cfg.Retries = 0
	c := NewClient(cfg)
	_, err := c.get(context.Background(), ts.URL)
	if err != ErrBlocked {
		t.Errorf("expected ErrBlocked, got %v", err)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.WikiBaseURL == "" {
		t.Error("WikiBaseURL should not be empty")
	}
	if cfg.ForumBaseURL == "" {
		t.Error("ForumBaseURL should not be empty")
	}
	if cfg.Rate <= 0 {
		t.Error("Rate should be positive")
	}
	if cfg.Retries <= 0 {
		t.Error("Retries should be positive")
	}
	if cfg.Timeout <= 0 {
		t.Error("Timeout should be positive")
	}
}

func TestNewClientDefaults(t *testing.T) {
	c := NewClient(DefaultConfig())
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
	if c.http == nil {
		t.Error("http client should not be nil")
	}
	if len(c.uaPool) == 0 {
		t.Error("UA pool should not be empty")
	}
}
