package adapter

import (
	"regexp"

	"github.com/example/hl7v2-message-router/internal/hl7/domain"
)

type Masker struct{ name *regexp.Regexp }

func NewMasker() *Masker {
	return &Masker{name: regexp.MustCompile(`(?i)(PID\|[^\r]*\|[^\r]*\|[^\r]*\|[^\r]*\|)[^|\r]+`)}
}
func (m *Masker) Mask(s string) string { return m.name.ReplaceAllString(s, "$1[REDACTED]") }

func (m *Masker) MaskMessage(message domain.Message) domain.Message {
	masked := message
	masked.Raw = m.Mask(masked.Raw)
	for i := range masked.Segments {
		if masked.Segments[i].Name != "PID" || len(masked.Segments[i].Fields) < 5 {
			continue
		}
		field := &masked.Segments[i].Fields[4]
		field.Value = "[REDACTED]"
		field.Repetitions = []string{"[REDACTED]"}
		field.Components = []string{"[REDACTED]"}
	}
	return masked
}
