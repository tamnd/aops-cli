package aops

import (
	"testing"
)

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "aops" {
		t.Errorf("Scheme = %q, want aops", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "aops" {
		t.Errorf("Identity.Binary = %q, want aops", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		in, typ, id string
	}{
		{"Prime number", "article", "Prime number"},
		{"2023 AMC 10A Problems", "article", "2023 AMC 10A Problems"},
		{"https://artofproblemsolving.com/wiki/index.php/Prime_number", "article", "Prime_number"},
	}
	for _, tc := range cases {
		typ, id, err := Domain{}.Classify(tc.in)
		if err != nil {
			t.Errorf("Classify(%q) error: %v", tc.in, err)
			continue
		}
		if typ != tc.typ {
			t.Errorf("Classify(%q) type = %q, want %q", tc.in, typ, tc.typ)
		}
		if id != tc.id {
			t.Errorf("Classify(%q) id = %q, want %q", tc.in, id, tc.id)
		}
	}
}

func TestLocateArticle(t *testing.T) {
	got, err := Domain{}.Locate("article", "Prime number")
	if err != nil {
		t.Fatalf("Locate: %v", err)
	}
	if got == "" {
		t.Error("URL should not be empty")
	}
}

func TestLocateUnknownType(t *testing.T) {
	_, err := Domain{}.Locate("bogus", "something")
	if err == nil {
		t.Error("expected error for unknown resource type")
	}
}

func TestWikiTitleFromInput(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Prime number", "Prime number"},
		{"Prime_number", "Prime number"},
		{"https://artofproblemsolving.com/wiki/index.php/Prime_number", "Prime_number"},
		{"https://artofproblemsolving.com/wiki/index.php?title=Prime+number", "Prime number"},
	}
	for _, tc := range cases {
		got := wikiTitleFromInput(tc.in)
		if got != tc.want {
			t.Errorf("wikiTitleFromInput(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
