package aops

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// WikiSearch calls the MediaWiki action=query&list=search endpoint.
func (c *Client) WikiSearch(ctx context.Context, query string, limit int) ([]WikiSearchResult, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	rawURL := c.cfg.WikiBaseURL + WikiAPIPath + "?" + url.Values{
		"action":      {"query"},
		"list":        {"search"},
		"srsearch":    {query},
		"srlimit":     {fmt.Sprintf("%d", limit)},
		"srnamespace": {"0"},
		"format":      {"json"},
		"formatversion": {"2"},
	}.Encode()

	body, err := c.get(ctx, rawURL)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Query struct {
			Search []struct {
				NS        int    `json:"ns"`
				Title     string `json:"title"`
				PageID    int    `json:"pageid"`
				Size      int    `json:"size"`
				Wordcount int    `json:"wordcount"`
				Snippet   string `json:"snippet"`
				Timestamp string `json:"timestamp"`
			} `json:"search"`
		} `json:"query"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("wiki search: decode: %w", err)
	}

	results := make([]WikiSearchResult, 0, len(resp.Query.Search))
	for _, s := range resp.Query.Search {
		ts, _ := time.Parse(time.RFC3339, s.Timestamp)
		results = append(results, WikiSearchResult{
			Title:     s.Title,
			NS:        s.NS,
			PageID:    s.PageID,
			Snippet:   stripHTMLTags(s.Snippet),
			Size:      s.Size,
			WordCount: s.Wordcount,
			Timestamp: ts,
			URL:       wikiPageURL(c.cfg.WikiBaseURL, s.Title),
		})
	}
	return results, nil
}

// WikiArticle calls the MediaWiki action=parse endpoint to get full article content.
func (c *Client) WikiArticle(ctx context.Context, title string) (*WikiArticle, error) {
	rawURL := c.cfg.WikiBaseURL + WikiAPIPath + "?" + url.Values{
		"action":      {"parse"},
		"page":        {title},
		"prop":        {"text|wikitext|sections|links"},
		"format":      {"json"},
		"formatversion": {"2"},
	}.Encode()

	body, err := c.get(ctx, rawURL)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Error *struct {
			Code string `json:"code"`
			Info string `json:"info"`
		} `json:"error"`
		Parse struct {
			Title  string `json:"title"`
			PageID int    `json:"pageid"`
			RevID  int    `json:"revid"`
			Text   string `json:"text"`
			Wikitext string `json:"wikitext"`
			Sections []struct {
				Index  string `json:"index"`
				Line   string `json:"line"`
				Anchor string `json:"anchor"`
				Level  string `json:"level"`
			} `json:"sections"`
			Links []struct {
				NS    int    `json:"ns"`
				Title string `json:"title"`
			} `json:"links"`
		} `json:"parse"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("wiki article: decode: %w", err)
	}
	if resp.Error != nil {
		if resp.Error.Code == "missingtitle" || resp.Error.Code == "missing" {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("wiki article: %s: %s", resp.Error.Code, resp.Error.Info)
	}

	sections := make([]WikiSection, 0, len(resp.Parse.Sections))
	for _, s := range resp.Parse.Sections {
		lvl := 0
		fmt.Sscanf(s.Level, "%d", &lvl)
		sections = append(sections, WikiSection{
			Index:  s.Index,
			Line:   s.Line,
			Anchor: s.Anchor,
			Level:  lvl,
		})
	}

	links := make([]WikiLink, 0, len(resp.Parse.Links))
	for _, l := range resp.Parse.Links {
		links = append(links, WikiLink{NS: l.NS, Title: l.Title})
	}

	content := htmlToText(resp.Parse.Text)

	return &WikiArticle{
		Title:        resp.Parse.Title,
		PageID:       resp.Parse.PageID,
		NS:           0,
		RevisionID:   resp.Parse.RevID,
		Content:      content,
		Wikitext:     resp.Parse.Wikitext,
		Sections:     sections,
		Links:        links,
		ContentModel: "wikitext",
		URL:          wikiPageURL(c.cfg.WikiBaseURL, resp.Parse.Title),
		FetchedAt:    time.Now().UTC(),
	}, nil
}

// WikiSuggest calls the MediaWiki action=opensearch endpoint.
func (c *Client) WikiSuggest(ctx context.Context, prefix string, limit int) ([]WikiSuggestion, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	rawURL := c.cfg.WikiBaseURL + WikiAPIPath + "?" + url.Values{
		"action": {"opensearch"},
		"search": {prefix},
		"limit":  {fmt.Sprintf("%d", limit)},
		"format": {"json"},
	}.Encode()

	body, err := c.get(ctx, rawURL)
	if err != nil {
		return nil, err
	}

	// Opensearch returns [query, [titles], [descs], [urls]]
	var raw []json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("wiki suggest: decode: %w", err)
	}
	if len(raw) < 4 {
		return nil, nil
	}

	var titles, descs, urls []string
	_ = json.Unmarshal(raw[1], &titles)
	_ = json.Unmarshal(raw[2], &descs)
	_ = json.Unmarshal(raw[3], &urls)

	results := make([]WikiSuggestion, 0, len(titles))
	for i, t := range titles {
		desc := ""
		if i < len(descs) {
			desc = descs[i]
		}
		u := ""
		if i < len(urls) {
			u = urls[i]
		}
		if u == "" {
			u = wikiPageURL(c.cfg.WikiBaseURL, t)
		}
		results = append(results, WikiSuggestion{Title: t, Description: desc, URL: u})
	}
	return results, nil
}

// WikiRaw fetches raw wikitext via /wiki/index.php?title=<title>&action=raw.
func (c *Client) WikiRaw(ctx context.Context, title string) (string, error) {
	rawURL := c.cfg.WikiBaseURL + WikiRawPath + "?" + url.Values{
		"title":  {title},
		"action": {"raw"},
	}.Encode()
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// WikiSections calls action=parse&prop=sections and returns the section list.
func (c *Client) WikiSections(ctx context.Context, title string) ([]WikiSection, error) {
	art, err := c.WikiArticle(ctx, title)
	if err != nil {
		return nil, err
	}
	return art.Sections, nil
}

// WikiLinks calls action=parse&prop=links and returns wikilinks.
func (c *Client) WikiLinks(ctx context.Context, title string) ([]WikiLink, error) {
	art, err := c.WikiArticle(ctx, title)
	if err != nil {
		return nil, err
	}
	return art.Links, nil
}

// wikiPageURL builds the canonical wiki article URL.
func wikiPageURL(base, title string) string {
	return base + "/wiki/index.php/" + url.PathEscape(strings.ReplaceAll(title, " ", "_"))
}

var (
	htmlTagRE    = regexp.MustCompile(`<[^>]+>`)
	multiSpaceRE = regexp.MustCompile(`\s+`)
)

// stripHTMLTags removes HTML tags and normalizes whitespace.
func stripHTMLTags(s string) string {
	s = htmlTagRE.ReplaceAllString(s, " ")
	s = multiSpaceRE.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// htmlToText converts HTML response text to readable plain text.
func htmlToText(html string) string {
	// Remove script and style blocks.
	s := regexp.MustCompile(`(?i)<(script|style)[^>]*>.*?</(script|style)>`).ReplaceAllString(html, "")
	// Replace block-level tags with newlines.
	s = regexp.MustCompile(`(?i)</(p|div|li|h[1-6]|br|tr)>`).ReplaceAllString(s, "\n")
	s = regexp.MustCompile(`(?i)<br\s*/?>`).ReplaceAllString(s, "\n")
	// Strip remaining tags.
	s = htmlTagRE.ReplaceAllString(s, "")
	// Decode common HTML entities.
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", `"`)
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	// Normalize whitespace, keeping paragraph breaks.
	lines := strings.Split(s, "\n")
	var out []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		out = append(out, line)
	}
	s = strings.Join(out, "\n")
	// Collapse multiple blank lines.
	s = regexp.MustCompile(`\n{3,}`).ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}
