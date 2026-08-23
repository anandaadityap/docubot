package util

import (
	"fmt"
	"strings"
)

const (
	SlugMinLen = 3
	SlugMaxLen = 48
)

var reservedSlugs = map[string]struct{}{
	"admin":    {},
	"api":      {},
	"login":    {},
	"register": {},
	"b":        {},
	"embed":    {},
	"demo":     {},
	"healthz":  {},
	"assets":   {},
	"static":   {},
}

// Slugify normalizes a display name into a URL slug (lowercase, hyphens).
// It may return a string shorter than SlugMinLen (including empty).
func Slugify(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	var b strings.Builder
	b.Grow(len(s))
	prevDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		case r == ' ' || r == '_' || r == '-':
			if b.Len() > 0 && !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// IsReservedSlug reports whether slug is blocked for public bot URLs.
func IsReservedSlug(slug string) bool {
	_, ok := reservedSlugs[slug]
	return ok
}

// ValidSlug reports whether s is already a usable public slug.
func ValidSlug(s string) bool {
	if s != Slugify(s) {
		return false
	}
	n := len(s)
	if n < SlugMinLen || n > SlugMaxLen {
		return false
	}
	return !IsReservedSlug(s)
}

// AllocateSlug picks a unique slug from seed. Collision falls back to bot-{userID}, then -2, -3.
func AllocateSlug(seed string, userID int64, taken func(string) bool) string {
	s := Slugify(seed)
	if len(s) < SlugMinLen || IsReservedSlug(s) || len(s) > SlugMaxLen {
		s = "bot"
	}
	if !taken(s) {
		return s
	}
	base := fmt.Sprintf("bot-%d", userID)
	if len(base) > SlugMaxLen {
		base = base[:SlugMaxLen]
	}
	if !taken(base) {
		return base
	}
	for n := 2; n < 10000; n++ {
		cand := fmt.Sprintf("%s-%d", base, n)
		if len(cand) > SlugMaxLen {
			cand = fmt.Sprintf("bot-%d-%d", userID%100000, n)
		}
		if !taken(cand) {
			return cand
		}
	}
	return fmt.Sprintf("bot-%d-x", userID)
}
