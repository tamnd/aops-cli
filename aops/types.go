package aops

import "time"

// WikiSearchResult is one hit from the MediaWiki search API.
type WikiSearchResult struct {
	Title     string    `json:"title"`
	NS        int       `json:"ns"`
	PageID    int       `json:"page_id"`
	Snippet   string    `json:"snippet"`
	Size      int       `json:"size"`
	WordCount int       `json:"word_count"`
	Timestamp time.Time `json:"timestamp"`
	URL       string    `json:"url"`
}

// WikiSection is a section heading inside a wiki article.
type WikiSection struct {
	Index  string `json:"index"`
	Line   string `json:"line"`
	Anchor string `json:"anchor"`
	Level  int    `json:"level"`
}

// WikiLink is a wikilink reference from a parsed article.
type WikiLink struct {
	NS    int    `json:"ns"`
	Title string `json:"title"`
}

// WikiArticle is a fully fetched wiki article.
type WikiArticle struct {
	Title        string        `json:"title"`
	PageID       int           `json:"page_id"`
	NS           int           `json:"ns"`
	RevisionID   int           `json:"revision_id"`
	Content      string        `json:"content"`
	Wikitext     string        `json:"wikitext"`
	Sections     []WikiSection `json:"sections"`
	Links        []WikiLink    `json:"links"`
	ContentModel string        `json:"content_model"`
	URL          string        `json:"url"`
	FetchedAt    time.Time     `json:"fetched_at"`
}

// WikiSuggestion is one result from the opensearch (autocomplete) API.
type WikiSuggestion struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

// Problem is one competition problem parsed from an AoPS wiki problem-set page.
type Problem struct {
	Contest       string   `json:"contest"`
	Year          int      `json:"year"`
	Number        int      `json:"number"`
	Statement     string   `json:"statement"`
	Answer        string   `json:"answer"`
	AnswerChoices []string `json:"answer_choices"`
	SolutionCount int      `json:"solution_count"`
	Solution      string   `json:"solution"`
	URL           string   `json:"url"`
	PageURL       string   `json:"page_url"`
}

// ForumTopic is one discussion thread header from the AoPS community forum.
type ForumTopic struct {
	TopicID      int       `json:"topic_id"`
	Title        string    `json:"title"`
	ForumID      int       `json:"forum_id"`
	ForumName    string    `json:"forum_name"`
	Author       string    `json:"author"`
	AuthorID     int       `json:"author_id"`
	ReplyCount   int       `json:"reply_count"`
	ViewCount    int       `json:"view_count"`
	LastPostTime time.Time `json:"last_post_time"`
	CreatedTime  time.Time `json:"created_time"`
	URL          string    `json:"url"`
}

// ForumPost is one post (reply) inside a forum topic.
type ForumPost struct {
	PostID      int       `json:"post_id"`
	TopicID     int       `json:"topic_id"`
	Author      string    `json:"author"`
	AuthorID    int       `json:"author_id"`
	PostNumber  int       `json:"post_number"`
	Content     string    `json:"content"`
	ThanksCount int       `json:"thanks_count"`
	PostedAt    time.Time `json:"posted_at"`
	IsHidden    bool      `json:"is_hidden"`
	URL         string    `json:"url"`
}
