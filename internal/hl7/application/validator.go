package application

import (
	"fmt"
	"github.com/example/hl7v2-message-router/internal/hl7/domain"
	"unicode/utf8"
)

type ValidationError struct{ Path, Reason string }

func (e ValidationError) Error() string { return fmt.Sprintf("%s: %s", e.Path, e.Reason) }
func Validate(m domain.Message) []error {
	var out []error
	if !utf8.ValidString(m.Raw) {
		out = append(out, ValidationError{"message", "invalid UTF-8"})
	}
	if len(m.Raw) > 1024*1024 {
		out = append(out, ValidationError{"message", "exceeds 1 MiB"})
	}
	if m.Version == "" {
		out = append(out, ValidationError{"MSH.12", "version required"})
	}
	return out
}
