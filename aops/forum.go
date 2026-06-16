package aops

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const communityBaseURL = "https://artofproblemsolving.com/community"

// topicURLRE matches AoPS community topic URLs like /community/c3h12345 or /community/c3h12345p67890.
var topicURLRE = regexp.MustCompile(`/community/c\d+h(\d+)`)

// parseTopicID extracts a numeric topic ID from a full AoPS URL or a bare integer string.
func ParseTopicID(input string) (int, error) {
	input = strings.TrimSpace(input)
	// Try bare integer first.
	if id, err := strconv.Atoi(input); err == nil && id > 0 {
		return id, nil
	}
	// Try URL pattern.
	m := topicURLRE.FindStringSubmatch(input)
	if m != nil {
		return strconv.Atoi(m[1])
	}
	return 0, fmt.Errorf("cannot parse topic ID from %q", input)
}

// forumAjaxURL returns the forum Ajax endpoint.
func (c *Client) forumAjaxURL() string {
	return c.cfg.ForumBaseURL + ForumAjaxPath
}

// ForumSearch searches forum topics via the Ajax API.
func (c *Client) ForumSearch(ctx context.Context, query string, limit int) ([]ForumTopic, error) {
	if limit <= 0 {
		limit = 20
	}
	formBody := url.Values{
		"a":             {"search_topics"},
		"search_string": {query},
		"limit":         {fmt.Sprintf("%d", limit)},
		"start":         {"0"},
	}.Encode()

	raw, err := c.post(ctx, c.forumAjaxURL(), formBody)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Status   string `json:"status"`
		ErrCode  string `json:"error_code"`
		ErrMsg   string `json:"error_msg"`
		Response struct {
			Topics []struct {
				TopicID      int    `json:"topicID"`
				TopicTitle   string `json:"topic_title"`
				ForumID      int    `json:"forumID"`
				ForumName    string `json:"forum_name"`
				Username     string `json:"username"`
				UserID       int    `json:"userID"`
				NumReplies   int    `json:"num_replies"`
				NumViews     int    `json:"num_views"`
				LastPostTime int64  `json:"last_post_time"`
				CreateTime   int64  `json:"create_time"`
			} `json:"topics"`
		} `json:"response"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("forum search: decode: %w", err)
	}
	if resp.Status != "ok" {
		if resp.ErrCode == "not_logged_in" {
			return nil, ErrBlocked
		}
		return nil, fmt.Errorf("forum search: %s", resp.ErrMsg)
	}

	topics := make([]ForumTopic, 0, len(resp.Response.Topics))
	for _, t := range resp.Response.Topics {
		topics = append(topics, ForumTopic{
			TopicID:      t.TopicID,
			Title:        t.TopicTitle,
			ForumID:      t.ForumID,
			ForumName:    t.ForumName,
			Author:       t.Username,
			AuthorID:     t.UserID,
			ReplyCount:   t.NumReplies,
			ViewCount:    t.NumViews,
			LastPostTime: time.Unix(t.LastPostTime, 0).UTC(),
			CreatedTime:  time.Unix(t.CreateTime, 0).UTC(),
			URL:          fmt.Sprintf("%s/c%dh%d", communityBaseURL, t.ForumID, t.TopicID),
		})
	}
	return topics, nil
}

// ForumTopicPosts fetches posts inside a topic via the Ajax API.
func (c *Client) ForumTopicPosts(ctx context.Context, topicID int, limit int) ([]ForumPost, error) {
	if limit <= 0 {
		limit = 30
	}
	if limit > 200 {
		limit = 200
	}
	formBody := url.Values{
		"a":             {"fetch_topic_contents"},
		"topic_id":      {fmt.Sprintf("%d", topicID)},
		"fetch_type":    {"recent"},
		"start_post_id": {"0"},
		"limit":         {fmt.Sprintf("%d", limit)},
		"user_id":       {"0"},
	}.Encode()

	raw, err := c.post(ctx, c.forumAjaxURL(), formBody)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Status   string `json:"status"`
		ErrCode  string `json:"error_code"`
		ErrMsg   string `json:"error_msg"`
		Response struct {
			Posts []struct {
				PostID         int    `json:"postID"`
				TopicID        int    `json:"topicID"`
				Username       string `json:"username"`
				UserID         int    `json:"userID"`
				PostCanonical  string `json:"post_canonical"`
				PostNumber     int    `json:"post_number"`
				NumThanks      int    `json:"num_thanks"`
				PostTime       int64  `json:"post_time"`
				IsHidden       int    `json:"isHidden"`
				ForumID        int    `json:"forumID"`
			} `json:"posts"`
		} `json:"response"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("forum topic: decode: %w", err)
	}
	if resp.Status != "ok" {
		if resp.ErrCode == "not_logged_in" {
			return nil, ErrBlocked
		}
		return nil, fmt.Errorf("forum topic: %s", resp.ErrMsg)
	}

	posts := make([]ForumPost, 0, len(resp.Response.Posts))
	for _, p := range resp.Response.Posts {
		posts = append(posts, ForumPost{
			PostID:      p.PostID,
			TopicID:     p.TopicID,
			Author:      p.Username,
			AuthorID:    p.UserID,
			PostNumber:  p.PostNumber,
			Content:     p.PostCanonical,
			ThanksCount: p.NumThanks,
			PostedAt:    time.Unix(p.PostTime, 0).UTC(),
			IsHidden:    p.IsHidden != 0,
			URL:         fmt.Sprintf("%s/c%dh%dp%d", communityBaseURL, p.ForumID, p.TopicID, p.PostID),
		})
	}
	return posts, nil
}
