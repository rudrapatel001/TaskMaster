package repository

import (
	"errors"
	"reflect"
	"testing"
)

func ptr(s string) *string { return &s }

func TestReorder(t *testing.T) {
	cases := []struct {
		name    string
		ids     []string
		moved   string
		before  *string
		want    []string
		wantErr error
	}{
		{"move to the very top", []string{"a", "b", "c"}, "c", ptr("a"), []string{"c", "a", "b"}, nil},
		{"move up one place", []string{"a", "b", "c"}, "c", ptr("b"), []string{"a", "c", "b"}, nil},
		{"move down one place", []string{"a", "b", "c"}, "a", ptr("c"), []string{"b", "a", "c"}, nil},
		{"move to the end", []string{"a", "b", "c"}, "a", nil, []string{"b", "c", "a"}, nil},
		{"moving the last one to the end changes nothing", []string{"a", "b", "c"}, "c", nil, []string{"a", "b", "c"}, nil},
		{"moving above the column that already follows it changes nothing", []string{"a", "b", "c"}, "a", ptr("b"), []string{"a", "b", "c"}, nil},
		{"a single column", []string{"a"}, "a", nil, []string{"a"}, nil},
		{"unknown column", []string{"a", "b"}, "zzz", nil, nil, ErrNotFound},
		{"unknown neighbour", []string{"a", "b"}, "a", ptr("zzz"), nil, ErrInvalidStatusNeighbour},
		{"a column cannot go before itself", []string{"a", "b"}, "a", ptr("a"), nil, ErrInvalidStatusNeighbour},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			original := append([]string(nil), c.ids...)

			got, err := reorder(c.ids, c.moved, c.before)

			if !errors.Is(err, c.wantErr) {
				t.Fatalf("error = %v, want %v", err, c.wantErr)
			}
			if c.wantErr == nil && !reflect.DeepEqual(got, c.want) {
				t.Fatalf("order = %v, want %v", got, c.want)
			}
			if !reflect.DeepEqual(c.ids, original) {
				t.Fatalf("the input slice was modified: %v, was %v", c.ids, original)
			}
		})
	}
}
