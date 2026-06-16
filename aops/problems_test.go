package aops

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContestSlugs(t *testing.T) {
	slugs := ContestSlugs()
	if len(slugs) == 0 {
		t.Fatal("ContestSlugs should return at least one slug")
	}
	// All expected slugs should be present.
	expected := []string{"AMC8", "AMC10A", "AIME1", "AIME2", "USAMO", "IMO"}
	slugSet := make(map[string]bool)
	for _, s := range slugs {
		slugSet[s] = true
	}
	for _, e := range expected {
		if !slugSet[e] {
			t.Errorf("expected slug %q not found", e)
		}
	}
}

func TestContestDefaultYear(t *testing.T) {
	if ContestDefaultYear("AMC10A") <= 0 {
		t.Error("AMC10A default year should be positive")
	}
	if ContestDefaultYear("BOGUS") != 0 {
		t.Error("unknown slug should return 0")
	}
}

func TestParseProblemsFromWikitext(t *testing.T) {
	wikitext := `
== Problem 1 ==
{{AMC10A Problems|year=2023|num-b=1|num-a=2}}

Cities A and B are 45 miles apart.
<math>\textbf{(A)}\ 2\qquad\textbf{(B)}\ 3\qquad\textbf{(C)}\ 4\qquad\textbf{(D)}\ 5\qquad\textbf{(E)}\ 6</math>

[[2023 AMC 10A Problems/Problem 1|Solution]]

== Solution 1 ==
Since the distance is 45 miles...

== Problem 2 ==
What is the sum of the first 10 positive integers?

[[2023 AMC 10A Problems/Problem 2|Solution]]
`
	problems := parseProblemsFromWikitext(
		"2023 AMC 10A Problems",
		wikitext,
		"AMC 10A",
		2023,
		"https://artofproblemsolving.com/wiki/index.php/2023_AMC_10A_Problems",
		"https://artofproblemsolving.com",
		false,
	)
	if len(problems) != 2 {
		t.Fatalf("want 2 problems, got %d", len(problems))
	}
	if problems[0].Number != 1 {
		t.Errorf("problem[0].number = %d, want 1", problems[0].Number)
	}
	if problems[0].Contest != "AMC 10A" {
		t.Errorf("contest = %q", problems[0].Contest)
	}
	if problems[0].Year != 2023 {
		t.Errorf("year = %d", problems[0].Year)
	}
	if problems[0].Statement == "" {
		t.Error("statement should not be empty")
	}
	if problems[0].URL == "" {
		t.Error("URL should not be empty")
	}
	if problems[0].PageURL == "" {
		t.Error("page URL should not be empty")
	}
	if problems[1].Number != 2 {
		t.Errorf("problem[1].number = %d, want 2", problems[1].Number)
	}
}

func TestParseProblemsWithSolutions(t *testing.T) {
	wikitext := `
== Problem 1 ==
Find all primes less than 10.

[[Test Problems/Problem 1|Solution]]

== Solution 1 ==
The primes are 2, 3, 5, 7.

== Problem 2 ==
What is 1+1?
`
	problems := parseProblemsFromWikitext(
		"Test Problems", wikitext, "Test", 2023,
		"https://example.com", "https://example.com", true,
	)
	if len(problems) != 2 {
		t.Fatalf("want 2 problems, got %d", len(problems))
	}
	if problems[0].Solution == "" {
		t.Error("solution should be included when includeSolutions=true")
	}
	if problems[0].SolutionCount < 1 {
		t.Errorf("solution_count = %d, want >= 1", problems[0].SolutionCount)
	}
}

func TestParseProblemsEmpty(t *testing.T) {
	problems := parseProblemsFromWikitext("Empty", "", "Test", 2023, "", "", false)
	if len(problems) != 0 {
		t.Errorf("empty wikitext should yield 0 problems, got %d", len(problems))
	}
}

func TestContestProblemsUnknown(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Rate = 0
	c := NewClient(cfg)
	_, err := c.ContestProblems(context.Background(), "BOGUS", 2023, false)
	if err == nil {
		t.Error("expected error for unknown contest")
	}
}

func TestContestProblems(t *testing.T) {
	wikitext := `
== Problem 1 ==
Find all prime numbers less than 10.

[[2022 USAMO Problems/Problem 1|Solution]]

== Problem 2 ==
Prove that there are infinitely many primes.

[[2022 USAMO Problems/Problem 2|Solution]]
`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, wikitext)
	}))
	defer ts.Close()

	cfg := DefaultConfig()
	cfg.WikiBaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 0
	c := NewClient(cfg)

	problems, err := c.ContestProblems(context.Background(), "USAMO", 2022, false)
	if err != nil {
		t.Fatalf("ContestProblems: %v", err)
	}
	if len(problems) != 2 {
		t.Fatalf("want 2 problems, got %d", len(problems))
	}
	if problems[0].Contest != "USAMO" {
		t.Errorf("contest = %q", problems[0].Contest)
	}
	if problems[0].Year != 2022 {
		t.Errorf("year = %d", problems[0].Year)
	}
}

func TestCleanWikitext(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Simple text.", "Simple text."},
		{"'''Bold''' text", "Bold text"},
		{"''italic''", "italic"},
		{"{{template}}", ""},
		{"[[Link|Label]]", "Label"},
		{"[[Link]]", "Link"},
		{"<math>x^2 + y^2</math>", "x^2 + y^2"},
	}
	for _, tc := range cases {
		got := cleanWikitext(tc.in)
		if got != tc.want {
			t.Errorf("cleanWikitext(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseTopicID(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"12345", 12345},
		{"https://artofproblemsolving.com/community/c3h12345", 12345},
		{"https://artofproblemsolving.com/community/c3h12345p67890", 12345},
		{"/community/c181h99999", 99999},
	}
	for _, tc := range cases {
		got, err := ParseTopicID(tc.in)
		if err != nil {
			t.Errorf("ParseTopicID(%q): %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseTopicID(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestParseTopicIDInvalid(t *testing.T) {
	_, err := ParseTopicID("not-a-valid-id")
	if err == nil {
		t.Error("expected error for invalid topic ID")
	}
}
