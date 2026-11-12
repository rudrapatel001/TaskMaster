package repository

import "testing"

func TestParseIdentifier(t *testing.T) {
	cases := []struct {
		in         string
		wantKey    string
		wantNumber int
		wantOK     bool
	}{
		{"WEB-12", "WEB", 12, true},
		{"web-12", "WEB", 12, true}, // typed in lowercase
		{"Api2-7", "API2", 7, true}, // digits allowed after the first letter
		{"AB-1", "AB", 1, true},
		{"WEB-0", "WEB", 0, true}, // matches the shape; simply finds nothing
		{"WEB", "", 0, false},     // no number
		{"WEB-", "", 0, false},
		{"-12", "", 0, false},
		{"1WEB-12", "", 0, false},       // keys start with a letter
		{"A-12", "", 0, false},          // keys have at least two characters
		{"ABCDEFGHIJK-1", "", 0, false}, // keys have at most ten
		{"web 12", "", 0, false},
		{"fix login", "", 0, false},
		{"WEB-12x", "", 0, false},
		{"WEB-1234567890", "", 0, false}, // more digits than a task number can have
		{"", "", 0, false},
	}
	for _, c := range cases {
		key, number, ok := parseIdentifier(c.in)
		if ok != c.wantOK || key != c.wantKey || number != c.wantNumber {
			t.Errorf("parseIdentifier(%q) = (%q, %d, %v), want (%q, %d, %v)",
				c.in, key, number, ok, c.wantKey, c.wantNumber, c.wantOK)
		}
	}
}
