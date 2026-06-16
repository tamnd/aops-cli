package aops

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestForumSearch(t *testing.T) {
	fixture := `{
	  "status": "ok",
	  "response": {
	    "topics": [
	      {
	        "topicID": 12345,
	        "topic_title": "2023 AMC 10A Problem 12 solution",
	        "forumID": 3,
	        "forum_name": "AMC, AIME, and ARML",
	        "username": "mathwiz2023",
	        "userID": 98765,
	        "num_replies": 14,
	        "num_views": 892,
	        "last_post_time": 1699720800,
	        "create_time": 1699520700
	      }
	    ]
	  }
	}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fixture)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	topics, err := c.ForumSearch(context.Background(), "AMC 10A 2023", 20)
	if err != nil {
		t.Fatalf("ForumSearch: %v", err)
	}
	if len(topics) != 1 {
		t.Fatalf("want 1 topic, got %d", len(topics))
	}
	if topics[0].TopicID != 12345 {
		t.Errorf("topic_id = %d, want 12345", topics[0].TopicID)
	}
	if topics[0].Title != "2023 AMC 10A Problem 12 solution" {
		t.Errorf("title = %q", topics[0].Title)
	}
	if topics[0].Author != "mathwiz2023" {
		t.Errorf("author = %q", topics[0].Author)
	}
	if topics[0].ReplyCount != 14 {
		t.Errorf("reply_count = %d", topics[0].ReplyCount)
	}
	if topics[0].URL == "" {
		t.Error("URL should not be empty")
	}
	if topics[0].LastPostTime.IsZero() {
		t.Error("last_post_time should not be zero")
	}
}

func TestForumSearchBlocked(t *testing.T) {
	fixture := `{"status":"error","error_code":"not_logged_in","error_msg":"You must be logged in."}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fixture)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.ForumSearch(context.Background(), "olympiad", 20)
	if err != ErrBlocked {
		t.Errorf("expected ErrBlocked, got %v", err)
	}
}

func TestForumSearchEmpty(t *testing.T) {
	fixture := `{"status":"ok","response":{"topics":[]}}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fixture)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	topics, err := c.ForumSearch(context.Background(), "xyzzy_nonexistent", 20)
	if err != nil {
		t.Fatalf("ForumSearch: %v", err)
	}
	if len(topics) != 0 {
		t.Errorf("want 0 topics, got %d", len(topics))
	}
}

func TestForumTopicPosts(t *testing.T) {
	fixture := `{
	  "status": "ok",
	  "response": {
	    "posts": [
	      {
	        "postID": 567890,
	        "topicID": 12345,
	        "username": "mathwiz2023",
	        "userID": 98765,
	        "post_canonical": "Here is my solution using Cauchy-Schwarz...",
	        "post_number": 1,
	        "num_thanks": 45,
	        "post_time": 1699520760,
	        "isHidden": 0,
	        "forumID": 3
	      },
	      {
	        "postID": 567891,
	        "topicID": 12345,
	        "username": "proofmaster",
	        "userID": 11111,
	        "post_canonical": "Alternative approach using AM-GM...",
	        "post_number": 2,
	        "num_thanks": 12,
	        "post_time": 1699521000,
	        "isHidden": 0,
	        "forumID": 3
	      }
	    ]
	  }
	}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fixture)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	posts, err := c.ForumTopicPosts(context.Background(), 12345, 30)
	if err != nil {
		t.Fatalf("ForumTopicPosts: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("want 2 posts, got %d", len(posts))
	}
	if posts[0].PostID != 567890 {
		t.Errorf("post_id = %d, want 567890", posts[0].PostID)
	}
	if posts[0].Author != "mathwiz2023" {
		t.Errorf("author = %q", posts[0].Author)
	}
	if posts[0].ThanksCount != 45 {
		t.Errorf("thanks_count = %d", posts[0].ThanksCount)
	}
	if posts[0].Content == "" {
		t.Error("content should not be empty")
	}
	if posts[0].URL == "" {
		t.Error("URL should not be empty")
	}
	if posts[0].IsHidden {
		t.Error("is_hidden should be false")
	}
	if posts[0].PostedAt.IsZero() {
		t.Error("posted_at should not be zero")
	}
}

func TestForumTopicPostsBlocked(t *testing.T) {
	fixture := `{"status":"error","error_code":"not_logged_in"}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fixture)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.ForumTopicPosts(context.Background(), 99999, 30)
	if err != ErrBlocked {
		t.Errorf("expected ErrBlocked, got %v", err)
	}
}
