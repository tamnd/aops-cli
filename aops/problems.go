package aops

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// contestMeta holds wiki page naming and problem count per contest slug.
type contestMeta struct {
	// wikiName builds the page title from a year, e.g. "2023 AMC 10A Problems".
	wikiName    func(year int) string
	displayName string
	probCount   int
	defaultYear int
}

// contests maps CLI slug to contest metadata.
var contests = map[string]contestMeta{
	"AMC8": {
		wikiName:    func(y int) string { return fmt.Sprintf("%d AMC 8 Problems", y) },
		displayName: "AMC 8",
		probCount:   25,
		defaultYear: 2024,
	},
	"AMC10A": {
		wikiName:    func(y int) string { return fmt.Sprintf("%d AMC 10A Problems", y) },
		displayName: "AMC 10A",
		probCount:   30,
		defaultYear: 2024,
	},
	"AMC10B": {
		wikiName:    func(y int) string { return fmt.Sprintf("%d AMC 10B Problems", y) },
		displayName: "AMC 10B",
		probCount:   30,
		defaultYear: 2024,
	},
	"AMC12A": {
		wikiName:    func(y int) string { return fmt.Sprintf("%d AMC 12A Problems", y) },
		displayName: "AMC 12A",
		probCount:   30,
		defaultYear: 2024,
	},
	"AMC12B": {
		wikiName:    func(y int) string { return fmt.Sprintf("%d AMC 12B Problems", y) },
		displayName: "AMC 12B",
		probCount:   30,
		defaultYear: 2024,
	},
	"AIME1": {
		wikiName:    func(y int) string { return fmt.Sprintf("%d AIME I Problems", y) },
		displayName: "AIME I",
		probCount:   15,
		defaultYear: 2024,
	},
	"AIME2": {
		wikiName:    func(y int) string { return fmt.Sprintf("%d AIME II Problems", y) },
		displayName: "AIME II",
		probCount:   15,
		defaultYear: 2024,
	},
	"USAMO": {
		wikiName:    func(y int) string { return fmt.Sprintf("%d USAMO Problems", y) },
		displayName: "USAMO",
		probCount:   6,
		defaultYear: 2024,
	},
	"IMO": {
		wikiName:    func(y int) string { return fmt.Sprintf("%d IMO Problems", y) },
		displayName: "IMO",
		probCount:   6,
		defaultYear: 2024,
	},
}

// ContestSlugs returns the sorted list of supported contest slugs.
func ContestSlugs() []string {
	return []string{"AMC8", "AMC10A", "AMC10B", "AMC12A", "AMC12B", "AIME1", "AIME2", "USAMO", "IMO"}
}

// ContestDefaultYear returns the default year for a contest slug.
// Returns 0 if the slug is unknown.
func ContestDefaultYear(slug string) int {
	if m, ok := contests[slug]; ok {
		return m.defaultYear
	}
	return 0
}

// ContestProblems fetches all problems for a given contest and year.
// It reads the master problem-list wikitext page and parses individual problems.
func (c *Client) ContestProblems(ctx context.Context, contest string, year int, includeSolutions bool) ([]Problem, error) {
	meta, ok := contests[strings.ToUpper(contest)]
	if !ok {
		return nil, fmt.Errorf("unknown contest %q; supported: %s", contest, strings.Join(ContestSlugs(), ", "))
	}
	if year <= 0 {
		year = meta.defaultYear
	}

	pageTitle := meta.wikiName(year)
	pageURL := wikiPageURL(c.cfg.WikiBaseURL, pageTitle)

	wikitext, err := c.WikiRaw(ctx, pageTitle)
	if err != nil {
		return nil, fmt.Errorf("contest problems %s %d: %w", contest, year, err)
	}

	problems := parseProblemsFromWikitext(pageTitle, wikitext, meta.displayName, year, pageURL, c.cfg.WikiBaseURL, includeSolutions)
	return problems, nil
}

// parseProblemsFromWikitext splits a contest page's wikitext into individual
// Problem records. It handles both AMC (multiple choice) and USAMO/IMO (open-ended) formats.
func parseProblemsFromWikitext(pageTitle, wikitext, contest string, year int, pageURL, baseURL string, includeSolutions bool) []Problem {
	// Split by == Problem N == headings.
	probHeadingRE := regexp.MustCompile(`(?m)^==\s*Problem\s+(\d+)\s*==`)
	solHeadingRE := regexp.MustCompile(`(?m)^==\s*Solution`)

	matches := probHeadingRE.FindAllStringSubmatchIndex(wikitext, -1)
	if len(matches) == 0 {
		return nil
	}

	var problems []Problem
	for i, m := range matches {
		numStr := wikitext[m[2]:m[3]]
		num, _ := strconv.Atoi(numStr)

		// Text from end of heading to next heading (or EOF).
		start := m[1]
		end := len(wikitext)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		section := wikitext[start:end]

		// Split at solution heading.
		solIdx := solHeadingRE.FindStringIndex(section)
		stmtPart := section
		solPart := ""
		solCount := 0
		if solIdx != nil {
			stmtPart = section[:solIdx[0]]
			if includeSolutions {
				solPart = section[solIdx[0]:]
				// Count solution subheadings.
				solCount = len(regexp.MustCompile(`(?m)^==+\s*Solution`).FindAllString(section, -1))
			} else {
				solCount = len(regexp.MustCompile(`(?m)^==+\s*Solution`).FindAllString(section, -1))
			}
		}

		// Strip wikitext markup from statement.
		stmt := cleanWikitext(stmtPart)

		// Extract answer choices for AMC-style problems.
		var choices []string
		answer := ""
		choices = extractAnswerChoices(stmtPart)

		sol := ""
		if includeSolutions && solPart != "" {
			sol = cleanWikitext(solPart)
		}

		probURL := baseURL + "/wiki/index.php/" + url.PathEscape(strings.ReplaceAll(
			pageTitle+"/Problem_"+numStr, " ", "_"))

		problems = append(problems, Problem{
			Contest:       contest,
			Year:          year,
			Number:        num,
			Statement:     stmt,
			Answer:        answer,
			AnswerChoices: choices,
			SolutionCount: solCount,
			Solution:      sol,
			URL:           probURL,
			PageURL:       pageURL,
		})
	}
	return problems
}

var (
	wikitextTemplateRE   = regexp.MustCompile(`\{\{[^}]*\}\}`)
	wikitextCategoryRE   = regexp.MustCompile(`\[\[Category:[^\]]*\]\]`)
	wikitextLinkRE       = regexp.MustCompile(`\[\[([^\]|]*)\|?([^\]]*)\]\]`)
	wikitextBoldRE       = regexp.MustCompile(`'''([^']*)'''`)
	wikitextItalicRE     = regexp.MustCompile(`''([^']*)''`)
	wikitextHeadingRE    = regexp.MustCompile(`(?m)^={2,6}[^=\n]+=+\s*$`)
	wikitextMathRE       = regexp.MustCompile(`(?s)<math>(.*?)</math>`)
	wikitextRefRE        = regexp.MustCompile(`(?s)<ref[^>]*>.*?</ref>`)
	answerChoiceRE       = regexp.MustCompile(`\\textbf\{\(([A-E])\)\}[\\,\s]*([^\\$\n]+)`)
	answerChoiceAltRE    = regexp.MustCompile(`\(([A-E])\)\s*\\?[\s]*([^()\n$]+)`)
)

// cleanWikitext strips wikitext markup and returns readable plain text.
func cleanWikitext(s string) string {
	// Remove <ref> tags.
	s = wikitextRefRE.ReplaceAllString(s, "")
	// Remove category links.
	s = wikitextCategoryRE.ReplaceAllString(s, "")
	// Remove templates {{...}}.
	s = wikitextTemplateRE.ReplaceAllString(s, "")
	// Simplify wikilinks [[target|label]] or [[target]].
	s = wikitextLinkRE.ReplaceAllStringFunc(s, func(m string) string {
		parts := wikitextLinkRE.FindStringSubmatch(m)
		if len(parts) < 3 || parts[2] != "" {
			if len(parts) >= 3 {
				return parts[2]
			}
		}
		if len(parts) >= 2 {
			return parts[1]
		}
		return ""
	})
	// Convert <math>...</math> to the raw math text.
	s = wikitextMathRE.ReplaceAllStringFunc(s, func(m string) string {
		inner := wikitextMathRE.FindStringSubmatch(m)
		if len(inner) >= 2 {
			return inner[1]
		}
		return ""
	})
	// Remove bold/italic markup.
	s = wikitextBoldRE.ReplaceAllString(s, "$1")
	s = wikitextItalicRE.ReplaceAllString(s, "$1")
	// Remove remaining headings.
	s = wikitextHeadingRE.ReplaceAllString(s, "")
	// Normalize whitespace.
	lines := strings.Split(s, "\n")
	var out []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return strings.TrimSpace(strings.Join(out, " "))
}

// extractAnswerChoices pulls AMC-style answer choices from wikitext.
// Supports both \textbf{(A)} and plain (A) formats.
func extractAnswerChoices(s string) []string {
	// Try \textbf{(A)} format in math blocks.
	mathBlocks := wikitextMathRE.FindAllStringSubmatch(s, -1)
	for _, mb := range mathBlocks {
		if len(mb) < 2 {
			continue
		}
		math := mb[1]
		matches := answerChoiceRE.FindAllStringSubmatch(math, -1)
		if len(matches) > 0 {
			choices := make([]string, 0, len(matches))
			for _, m := range matches {
				if len(m) >= 3 {
					choices = append(choices, fmt.Sprintf("(%s) %s", m[1], strings.TrimSpace(m[2])))
				}
			}
			if len(choices) > 0 {
				return choices
			}
		}
	}
	return nil
}
