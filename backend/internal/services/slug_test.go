package services

import (
	"regexp"
	"strings"
	"testing"
)

// The same rule the database enforces (see migration 2).
var slugRule = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Marketing Team":         "marketing-team",
		"  Rudra's  Team!!  ":    "rudras-team",
		"Q3 -- Planning (2026)":  "q3-planning-2026",
		"---":                    "workspace",
		"ab":                     "workspace",
		"રુદ્ર ટીમ":              "workspace",
		"Café Crew":              "caf-crew",
		strings.Repeat("a", 100): strings.Repeat("a", 40),

		// Long names are cut at 40 characters, even in the middle of a word.
		strings.Repeat("ab ", 30): "ab-ab-ab-ab-ab-ab-ab-ab-ab-ab-ab-ab-ab-a",

		// If the cut lands exactly on a hyphen, that hyphen is removed.
		strings.Repeat("a", 39) + " b": strings.Repeat("a", 39),
	}
	for in, want := range cases {
		got := slugify(in)
		if got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
		if !slugRule.MatchString(got) || len(got) < 3 || len(got) > 50 {
			t.Errorf("slugify(%q) = %q breaks the database's slug rules", in, got)
		}
	}
}
