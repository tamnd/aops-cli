package aops

import (
	"errors"
	"fmt"
)

// ErrBlocked is returned when AoPS or Cloudflare refuses the request.
var ErrBlocked = errors.New("aops: blocked (Cloudflare challenge or auth required)")

// ErrNotFound is returned when the requested wiki page or topic does not exist.
var ErrNotFound = errors.New("aops: not found")

// ErrRateLimited is returned after exhausting retries on HTTP 429.
var ErrRateLimited = errors.New("aops: rate limited (HTTP 429)")

// HTTPError carries the status code for unexpected responses.
type HTTPError struct {
	StatusCode int
	URL        string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("aops: HTTP %d from %s", e.StatusCode, e.URL)
}
