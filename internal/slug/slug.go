package slug

import (
	"crypto/rand"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const (
	fallback     = "article"
	suffixLength = 6
	alphabet     = "abcdefghijklmnopqrstuvwxyz0123456789"
)

func Make(title string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	s, _, err := transform.String(t, strings.ToLower(title))
	if err != nil {
		s = strings.ToLower(title)
	}
	s = strings.ReplaceAll(s, "đ", "d")

	var b strings.Builder
	dash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	if out := strings.TrimSuffix(b.String(), "-"); out != "" {
		return out
	}
	return fallback
}

func WithSuffix(base string) string {
	buf := make([]byte, suffixLength)
	_, _ = rand.Read(buf)
	for i, v := range buf {
		buf[i] = alphabet[int(v)%len(alphabet)]
	}
	return base + "-" + string(buf)
}
