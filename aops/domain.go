package aops

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes aops as a kit Domain: a driver that a multi-domain
// host (ant) enables with a single blank import,
//
//	import _ "github.com/tamnd/aops-cli/aops"
//
// The Domain registers the client factory and all operations so the
// standalone binary and a kit host share one source of truth.
func init() { kit.Register(Domain{}) }

// Domain is the aops driver.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against,
// and the identity reused for the binary help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "aops",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "aops",
			Short:  "Browse Art of Problem Solving wiki and forums",
			Long: `Browse Art of Problem Solving wiki and forums

aops reads public artofproblemsolving.com data over HTTPS and prints
structured records that pipe into the rest of your tools. No API key required.`,
			Site: Host,
			Repo: "https://github.com/tamnd/aops-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{
		Name:    "wiki-search",
		Group:   "wiki",
		List:    true,
		Summary: "Search AoPS wiki articles",
		Args:    []kit.Arg{{Name: "query", Help: "search query"}},
	}, handleWikiSearch)

	kit.Handle(app, kit.OpMeta{
		Name:    "wiki-article",
		Group:   "wiki",
		Single:  true,
		Summary: "Fetch an AoPS wiki article",
		Args:    []kit.Arg{{Name: "title", Help: "article title"}},
	}, handleWikiArticle)

	kit.Handle(app, kit.OpMeta{
		Name:    "wiki-suggest",
		Group:   "wiki",
		List:    true,
		Summary: "Autocomplete AoPS wiki article titles",
		Args:    []kit.Arg{{Name: "prefix", Help: "title prefix"}},
	}, handleWikiSuggest)

	kit.Handle(app, kit.OpMeta{
		Name:    "problems",
		Group:   "contest",
		List:    true,
		Summary: "List competition problems from the AoPS wiki",
		Args:    []kit.Arg{{Name: "contest", Help: "contest slug (AMC8, AMC10A, AIME1, USAMO, IMO, ...)"}},
	}, handleProblems)

	kit.Handle(app, kit.OpMeta{
		Name:    "forum-search",
		Group:   "forum",
		List:    true,
		Summary: "Search AoPS forum topics",
		Args:    []kit.Arg{{Name: "query", Help: "search query"}},
	}, handleForumSearch)

	kit.Handle(app, kit.OpMeta{
		Name:    "forum-topic",
		Group:   "forum",
		List:    true,
		Summary: "Fetch posts in an AoPS forum topic",
		Args:    []kit.Arg{{Name: "topic", Help: "topic ID or URL"}},
	}, handleForumTopic)
}

// newClient builds the client from the kit-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := NewClient(DefaultConfig())
	if cfg.UserAgent != "" {
		c.cfg.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.cfg.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.cfg.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.cfg.Timeout = cfg.Timeout
		c.http.Timeout = cfg.Timeout
	}
	return c, nil
}

// --- input structs ---

type wikiSearchIn struct {
	Query  string  `kit:"arg" help:"search query"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

type wikiArticleIn struct {
	Title  string  `kit:"arg" help:"article title"`
	Client *Client `kit:"inject"`
}

type wikiSuggestIn struct {
	Prefix string  `kit:"arg" help:"title prefix"`
	Limit  int     `kit:"flag,inherit" help:"max suggestions"`
	Client *Client `kit:"inject"`
}

type problemsIn struct {
	Contest string  `kit:"arg" help:"contest slug (AMC8, AMC10A, AIME1, USAMO, IMO)"`
	Limit   int     `kit:"flag,inherit" help:"max problems"`
	Client  *Client `kit:"inject"`
}

type forumSearchIn struct {
	Query  string  `kit:"arg" help:"search query"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

type forumTopicIn struct {
	Topic  string  `kit:"arg" help:"topic ID or URL"`
	Limit  int     `kit:"flag,inherit" help:"max posts"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func handleWikiSearch(ctx context.Context, in wikiSearchIn, emit func(*WikiSearchResult) error) error {
	n := in.Limit
	if n <= 0 {
		n = 10
	}
	results, err := in.Client.WikiSearch(ctx, in.Query, n)
	if err != nil {
		return mapErr(err)
	}
	for i := range results {
		if err := emit(&results[i]); err != nil {
			return err
		}
	}
	return nil
}

func handleWikiArticle(ctx context.Context, in wikiArticleIn, emit func(*WikiArticle) error) error {
	art, err := in.Client.WikiArticle(ctx, in.Title)
	if err != nil {
		return mapErr(err)
	}
	return emit(art)
}

func handleWikiSuggest(ctx context.Context, in wikiSuggestIn, emit func(*WikiSuggestion) error) error {
	n := in.Limit
	if n <= 0 {
		n = 10
	}
	results, err := in.Client.WikiSuggest(ctx, in.Prefix, n)
	if err != nil {
		return mapErr(err)
	}
	for i := range results {
		if err := emit(&results[i]); err != nil {
			return err
		}
	}
	return nil
}

func handleProblems(ctx context.Context, in problemsIn, emit func(*Problem) error) error {
	problems, err := in.Client.ContestProblems(ctx, in.Contest, 0, false)
	if err != nil {
		return mapErr(err)
	}
	n := in.Limit
	for i := range problems {
		if n > 0 && i >= n {
			break
		}
		if err := emit(&problems[i]); err != nil {
			return err
		}
	}
	return nil
}

func handleForumSearch(ctx context.Context, in forumSearchIn, emit func(*ForumTopic) error) error {
	n := in.Limit
	if n <= 0 {
		n = 20
	}
	topics, err := in.Client.ForumSearch(ctx, in.Query, n)
	if err != nil {
		return mapErr(err)
	}
	for i := range topics {
		if err := emit(&topics[i]); err != nil {
			return err
		}
	}
	return nil
}

func handleForumTopic(ctx context.Context, in forumTopicIn, emit func(*ForumPost) error) error {
	id, err := ParseTopicID(in.Topic)
	if err != nil {
		return errs.Usage("invalid topic: %s", err)
	}
	n := in.Limit
	if n <= 0 {
		n = 30
	}
	posts, err := in.Client.ForumTopicPosts(ctx, id, n)
	if err != nil {
		return mapErr(err)
	}
	for i := range posts {
		if err := emit(&posts[i]); err != nil {
			return err
		}
	}
	return nil
}

// --- Resolver ---

// Classify turns accepted input into (uriType, id).
func (Domain) Classify(input string) (uriType, id string, err error) {
	id = wikiTitleFromInput(input)
	if id == "" {
		return "", "", errs.Usage("unrecognized aops reference: %q", input)
	}
	return "article", id, nil
}

// Locate returns the canonical https URL for a (type, id) pair.
func (Domain) Locate(uriType, id string) (string, error) {
	switch uriType {
	case "article":
		return wikiPageURL(BaseURL, id), nil
	default:
		return "", errs.Usage("aops has no resource type %q", uriType)
	}
}

// mapErr converts library errors to kit errors with proper exit codes.
func mapErr(err error) error {
	switch {
	case errors.Is(err, ErrBlocked):
		return errs.NeedAuth("%s", err.Error())
	case errors.Is(err, ErrNotFound):
		return errs.NotFound("%s", err.Error())
	case errors.Is(err, ErrRateLimited):
		return errs.RateLimited("%s", err.Error())
	}
	return err
}

// wikiTitleFromInput extracts an article title from a full AoPS wiki URL or
// treats bare input as a title.
func wikiTitleFromInput(input string) string {
	input = strings.TrimSpace(input)
	if u, err := url.Parse(input); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		// Handle /wiki/index.php/Title or /wiki/index.php?title=Title
		path := u.Path
		if strings.Contains(path, "/wiki/index.php/") {
			return strings.TrimPrefix(path, "/wiki/index.php/")
		}
		if t := u.Query().Get("title"); t != "" {
			return t
		}
		return strings.Trim(path, "/")
	}
	return strings.ReplaceAll(input, "_", " ")
}
