// Package aops is the library behind the aops command line:
// the HTTP client, request shaping, and typed data models for
// artofproblemsolving.com.
package aops

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Host is the primary hostname.
const Host = "artofproblemsolving.com"

// BaseURL is the root for all requests.
const BaseURL = "https://" + Host

// Config holds runtime settings for the Client.
type Config struct {
	WikiBaseURL  string
	ForumBaseURL string
	UserAgent    string
	Rate         time.Duration
	Retries      int
	Timeout      time.Duration
	Cookie       string
}

// DefaultConfig returns polite, production-ready defaults.
func DefaultConfig() Config {
	return Config{
		WikiBaseURL:  WikiBaseURL,
		ForumBaseURL: ForumBaseURL,
		Rate:         DefaultRate,
		Retries:      DefaultRetries,
		Timeout:      DefaultTimeout,
	}
}

// Client talks to artofproblemsolving.com with pacing and retry.
type Client struct {
	cfg     Config
	http    *http.Client
	uaPool  []string
	mu      sync.Mutex
	lastReq time.Time
	uaIdx   int
}

// NewClient builds a Client from cfg.
func NewClient(cfg Config) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultTimeout
	}
	if cfg.Retries <= 0 {
		cfg.Retries = DefaultRetries
	}
	if cfg.Rate < 0 {
		cfg.Rate = 0
	}
	if cfg.WikiBaseURL == "" {
		cfg.WikiBaseURL = WikiBaseURL
	}
	if cfg.ForumBaseURL == "" {
		cfg.ForumBaseURL = ForumBaseURL
	}
	pool := userAgents
	return &Client{
		cfg:    cfg,
		http:   &http.Client{Timeout: cfg.Timeout},
		uaPool: pool,
		uaIdx:  rand.Intn(len(pool)),
	}
}

// ua returns the next User-Agent from the rotating pool, or the configured
// override when one is set.
func (c *Client) ua() string {
	if c.cfg.UserAgent != "" {
		return c.cfg.UserAgent
	}
	c.mu.Lock()
	ua := c.uaPool[c.uaIdx%len(c.uaPool)]
	c.uaIdx++
	c.mu.Unlock()
	return ua
}

// pace blocks until at least Rate has elapsed since the last request.
func (c *Client) pace() {
	if c.cfg.Rate <= 0 {
		return
	}
	c.mu.Lock()
	wait := c.cfg.Rate - time.Since(c.lastReq)
	c.mu.Unlock()
	if wait > 0 {
		time.Sleep(wait)
	}
	c.mu.Lock()
	c.lastReq = time.Now()
	c.mu.Unlock()
}

// isCloudflareChallenge reports whether body is a Cloudflare challenge page.
func isCloudflareChallenge(b []byte) bool {
	return bytes.Contains(b, []byte("Just a moment...")) ||
		bytes.Contains(b, []byte("cf-mitigated"))
}

// get performs a paced GET with retry. Returns ErrBlocked on Cloudflare
// challenge, ErrRateLimited after exhausting 429 retries.
func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryWait(attempt)):
			}
		}
		c.pace()
		body, retry, err := c.doGet(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) doGet(ctx context.Context, rawURL string) (body []byte, retry bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.ua())
	req.Header.Set("Accept", "application/json,text/html,*/*;q=0.9")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	if c.cfg.Cookie != "" {
		req.Header.Set("Cookie", c.cfg.Cookie)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}

	if isCloudflareChallenge(b) {
		return nil, false, ErrBlocked
	}

	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, true, ErrRateLimited
	case resp.StatusCode == http.StatusNotFound:
		return nil, false, ErrNotFound
	case resp.StatusCode == http.StatusForbidden:
		return nil, false, ErrBlocked
	case resp.StatusCode >= 500:
		return nil, true, &HTTPError{StatusCode: resp.StatusCode, URL: rawURL}
	case resp.StatusCode != http.StatusOK:
		return nil, false, &HTTPError{StatusCode: resp.StatusCode, URL: rawURL}
	}
	return b, false, nil
}

// post performs a paced POST with retry (for forum Ajax calls).
func (c *Client) post(ctx context.Context, rawURL, formBody string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryWait(attempt)):
			}
		}
		c.pace()
		out, retry, err := c.doPost(ctx, rawURL, formBody)
		if err == nil {
			return out, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("post %s: %w", rawURL, lastErr)
}

func (c *Client) doPost(ctx context.Context, rawURL, formBody string) (out []byte, retry bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, strings.NewReader(formBody))
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.ua())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	if c.cfg.Cookie != "" {
		req.Header.Set("Cookie", c.cfg.Cookie)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}

	if isCloudflareChallenge(b) {
		return nil, false, ErrBlocked
	}

	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, true, ErrRateLimited
	case resp.StatusCode == http.StatusForbidden:
		return nil, false, ErrBlocked
	case resp.StatusCode >= 500:
		return nil, true, &HTTPError{StatusCode: resp.StatusCode, URL: rawURL}
	case resp.StatusCode != http.StatusOK:
		return nil, false, &HTTPError{StatusCode: resp.StatusCode, URL: rawURL}
	}
	return b, false, nil
}

func retryWait(attempt int) time.Duration {
	d := time.Duration(attempt) * 2 * time.Second
	if d > 10*time.Second {
		return 10 * time.Second
	}
	return d
}
