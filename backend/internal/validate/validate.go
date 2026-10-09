package validate

import (
	"net/mail"
	"strings"
	"unicode/utf8"
)

// Validator collects one error message per field.
type Validator struct {
	fields map[string]string
}

func New() *Validator {
	return &Validator{fields: make(map[string]string)}
}

// Check records message against field when ok is false.
// If the field already has an error, the first one is kept, so list the most
// important check (usually "required") first.
func (v *Validator) Check(ok bool, field, message string) {
	if ok {
		return
	}
	if _, exists := v.fields[field]; !exists {
		v.fields[field] = message
	}
}

// Valid reports whether every check so far has passed.
func (v *Validator) Valid() bool {
	return len(v.fields) == 0
}

// Fields returns the collected field errors, keyed by field name.
func (v *Validator) Fields() map[string]string {
	return v.fields
}

// NotBlank is true if s contains something other than whitespace.
func NotBlank(s string) bool {
	return strings.TrimSpace(s) != ""
}

// MinLen is true if s has at least n characters (not bytes).
func MinLen(s string, n int) bool {
	return utf8.RuneCountInString(s) >= n
}

// MaxLen is true if s has at most n characters (not bytes).
func MaxLen(s string, n int) bool {
	return utf8.RuneCountInString(s) <= n
}

// Email is true if s looks like a plain email address such as a@b.com.
// It only checks the format; proving the address is real needs a confirmation email.
func Email(s string) bool {
	if len(s) > 254 {
		return false
	}
	addr, err := mail.ParseAddress(s)
	// ParseAddress also accepts "Name <a@b.com>"; require the bare address.
	return err == nil && addr.Address == s
}

// OneOf is true if s equals one of the allowed values.
func OneOf(s string, allowed ...string) bool {
	for _, a := range allowed {
		if s == a {
			return true
		}
	}
	return false
}
