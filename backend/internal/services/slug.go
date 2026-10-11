package services

import "strings"

const (
	maxSlugBase   = 40 // leaves room for a "-xxxx" suffix inside the 50-character limit
	minSlugLength = 3  // the database requires at least 3
)

// slugify turns a workspace name into a URL-friendly slug: lowercase letters
// and digits separated by single hyphens, for example "Rudra's Team!" -> "rudras-team".
func slugify(name string) string {
	var b strings.Builder
	lastWasHyphen := true // true at the start, so the slug never begins with a hyphen

	for _, r := range strings.ToLower(name) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastWasHyphen = false
		case r == '\'' || r == '’':
			// apostrophes are dropped: "rudra's" -> "rudras"
		case !lastWasHyphen:
			b.WriteByte('-')
			lastWasHyphen = true
		}
	}

	slug := strings.TrimRight(b.String(), "-")
	if len(slug) > maxSlugBase {
		slug = strings.TrimRight(slug[:maxSlugBase], "-")
	}
	if len(slug) < minSlugLength {
		return "workspace"
	}
	return slug
}
