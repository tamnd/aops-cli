package aops

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(ts *httptest.Server) *Client {
	cfg := DefaultConfig()
	cfg.WikiBaseURL = ts.URL
	cfg.ForumBaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 0
	cfg.Timeout = 0
	return NewClient(cfg)
}

func TestWikiSearch(t *testing.T) {
	fixture := `{
	  "batchcomplete": true,
	  "query": {
	    "search": [
	      {
	        "ns": 0,
	        "title": "Prime number",
	        "pageid": 100,
	        "size": 8200,
	        "wordcount": 1200,
	        "snippet": "A <span class=\"searchmatch\">prime</span> number is...",
	        "timestamp": "2024-01-10T08:22:00Z"
	      },
	      {
	        "ns": 0,
	        "title": "Prime factorization",
	        "pageid": 200,
	        "size": 4100,
	        "wordcount": 600,
	        "snippet": "Prime factorization is...",
	        "timestamp": "2024-01-08T10:00:00Z"
	      }
	    ]
	  }
	}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fixture)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	results, err := c.WikiSearch(context.Background(), "prime", 10)
	if err != nil {
		t.Fatalf("WikiSearch: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("want 2 results, got %d", len(results))
	}
	if results[0].Title != "Prime number" {
		t.Errorf("title = %q, want Prime number", results[0].Title)
	}
	if results[0].PageID != 100 {
		t.Errorf("page_id = %d, want 100", results[0].PageID)
	}
	// snippet should have HTML tags stripped.
	if results[0].Snippet == "" {
		t.Error("snippet should not be empty")
	}
	if results[0].URL == "" {
		t.Error("URL should not be empty")
	}
	// Timestamp should parse.
	if results[0].Timestamp.IsZero() {
		t.Error("timestamp should not be zero")
	}
}

func TestWikiSearchEmpty(t *testing.T) {
	fixture := `{"batchcomplete":true,"query":{"search":[]}}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fixture)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	results, err := c.WikiSearch(context.Background(), "xyzzy_nonexistent", 10)
	if err != nil {
		t.Fatalf("WikiSearch: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("want 0 results, got %d", len(results))
	}
}

func TestWikiArticle(t *testing.T) {
	fixture := `{
	  "parse": {
	    "title": "Pythagorean theorem",
	    "pageid": 555,
	    "revid": 12345,
	    "text": "<div class=\"mw-parser-output\"><p>The <b>Pythagorean theorem</b> states...</p></div>",
	    "wikitext": "The '''Pythagorean theorem''' states that...",
	    "sections": [
	      {"index": "1", "line": "Statement", "anchor": "Statement", "level": "2"}
	    ],
	    "links": [
	      {"ns": 0, "title": "Triangle", "exists": true}
	    ]
	  }
	}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fixture)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	art, err := c.WikiArticle(context.Background(), "Pythagorean theorem")
	if err != nil {
		t.Fatalf("WikiArticle: %v", err)
	}
	if art.Title != "Pythagorean theorem" {
		t.Errorf("title = %q", art.Title)
	}
	if art.PageID != 555 {
		t.Errorf("page_id = %d", art.PageID)
	}
	if art.RevisionID != 12345 {
		t.Errorf("revision_id = %d", art.RevisionID)
	}
	if art.Wikitext == "" {
		t.Error("wikitext should not be empty")
	}
	if art.URL == "" {
		t.Error("URL should not be empty")
	}
	if art.FetchedAt.IsZero() {
		t.Error("fetched_at should not be zero")
	}
	if len(art.Sections) != 1 {
		t.Errorf("sections: want 1, got %d", len(art.Sections))
	}
	if art.Sections[0].Line != "Statement" {
		t.Errorf("section line = %q", art.Sections[0].Line)
	}
	if len(art.Links) != 1 {
		t.Errorf("links: want 1, got %d", len(art.Links))
	}
	if art.Links[0].Title != "Triangle" {
		t.Errorf("link title = %q", art.Links[0].Title)
	}
}

func TestWikiArticleNotFound(t *testing.T) {
	fixture := `{"error":{"code":"missingtitle","info":"The page you specified doesn't exist."}}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fixture)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.WikiArticle(context.Background(), "Nonexistent_Page_XYZ")
	if err == nil {
		t.Fatal("expected error for missing page")
	}
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestWikiSuggest(t *testing.T) {
	fixture := `[
	  "2023 AMC",
	  ["2023 AMC 10A Problems", "2023 AMC 10B Problems", "2023 AMC 12A Problems"],
	  ["Competition problems", "Competition problems", "Competition problems"],
	  ["https://artofproblemsolving.com/wiki/index.php/2023_AMC_10A_Problems",
	   "https://artofproblemsolving.com/wiki/index.php/2023_AMC_10B_Problems",
	   "https://artofproblemsolving.com/wiki/index.php/2023_AMC_12A_Problems"]
	]`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fixture)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	results, err := c.WikiSuggest(context.Background(), "2023 AMC", 10)
	if err != nil {
		t.Fatalf("WikiSuggest: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("want 3 suggestions, got %d", len(results))
	}
	if results[0].Title != "2023 AMC 10A Problems" {
		t.Errorf("title = %q", results[0].Title)
	}
	if results[0].URL == "" {
		t.Error("URL should not be empty")
	}
}

func TestWikiSuggestEmpty(t *testing.T) {
	fixture := `["xyz",[],[],[]]`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fixture)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	results, err := c.WikiSuggest(context.Background(), "xyz", 10)
	if err != nil {
		t.Fatalf("WikiSuggest: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("want 0 suggestions, got %d", len(results))
	}
}

func TestWikiRaw(t *testing.T) {
	fixture := "== Problem 1 ==\nFind all prime numbers..."
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fixture)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	text, err := c.WikiRaw(context.Background(), "2023 AMC 10A Problems")
	if err != nil {
		t.Fatalf("WikiRaw: %v", err)
	}
	if text != fixture {
		t.Errorf("raw text = %q, want %q", text, fixture)
	}
}

func TestWikiBlocked(t *testing.T) {
	fixture := `<!DOCTYPE html><html><head><title>Just a moment...</title></head><body></body></html>`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fixture)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.WikiSearch(context.Background(), "prime", 10)
	if err != ErrBlocked {
		t.Errorf("expected ErrBlocked, got %v", err)
	}
}

func TestStripHTMLTags(t *testing.T) {
	cases := []struct{ in, want string }{
		{"<b>hello</b>", "hello"},
		{`A <span class="searchmatch">prime</span> number`, "A prime number"},
		{"no tags here", "no tags here"},
		{"  spaces  ", "spaces"},
	}
	for _, tc := range cases {
		got := stripHTMLTags(tc.in)
		if got != tc.want {
			t.Errorf("stripHTMLTags(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
