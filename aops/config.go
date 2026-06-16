package aops

import "time"

// Site constants.
const (
	WikiBaseURL  = "https://artofproblemsolving.com"
	ForumBaseURL = "https://artofproblemsolving.com"

	WikiAPIPath   = "/wiki/api.php"
	WikiRawPath   = "/wiki/index.php"
	ForumAjaxPath = "/m/community/ajax.php"

	DefaultRate    = 1 * time.Second
	DefaultTimeout = 30 * time.Second
	DefaultRetries = 3
)

// userAgents is a small pool of real browser User-Agent strings.
var userAgents = []string{
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:125.0) Gecko/20100101 Firefox/125.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_4_1) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4.1 Safari/605.1.15",
}
