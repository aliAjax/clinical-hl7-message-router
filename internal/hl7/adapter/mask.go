package adapter

import "regexp"

type Masker struct{ name *regexp.Regexp }

func NewMasker() *Masker {
	return &Masker{name: regexp.MustCompile(`(?i)(PID\|[^\r]*\|[^\r]*\|[^\r]*\|[^\r]*\|)[^|\r]+`)}
}
func (m *Masker) Mask(s string) string { return m.name.ReplaceAllString(s, "$1[REDACTED]") }
