package application

import (
	"fmt"
	"strings"

	"github.com/example/hl7v2-message-router/internal/hl7/domain"
)

type Separators struct {
	Field        string
	Component    string
	Repetition   string
	Escape       string
	Subcomponent string
}

func SeparatorsFrom(message domain.Message) Separators {
	out := Separators{Field: "|", Component: "^", Repetition: "~", Escape: "\\", Subcomponent: "&"}
	if len(message.Segments) == 0 || message.Segments[0].Name != "MSH" {
		return out
	}
	raw := message.Segments[0].Raw
	if len(raw) > 3 {
		out.Field = string(raw[3])
	}
	if len(raw) >= 8 {
		out.Component = string(raw[4])
		out.Repetition = string(raw[5])
		out.Escape = string(raw[6])
		out.Subcomponent = string(raw[7])
	}
	return out
}

func Serialize(message domain.Message) (string, error) {
	if len(message.Segments) == 0 {
		return "", fmt.Errorf("message has no segments")
	}
	separators := SeparatorsFrom(message)
	lines := make([]string, 0, len(message.Segments))
	for _, segment := range message.Segments {
		if len(segment.Name) != 3 {
			return "", fmt.Errorf("invalid segment name %q", segment.Name)
		}
		values := make([]string, 0, len(segment.Fields)+1)
		values = append(values, segment.Name)
		for _, field := range segment.Fields {
			if len(field.Repetitions) > 0 {
				values = append(values, strings.Join(field.Repetitions, separators.Repetition))
			} else {
				values = append(values, field.Value)
			}
		}
		line := strings.Join(values, separators.Field)
		if segment.Name == "MSH" && len(segment.Fields) > 0 {
			// MSH-1 is the field separator and is encoded directly after MSH.
			line = "MSH" + separators.Field + strings.Join(values[2:], separators.Field)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\r") + "\r", nil
}

func Escape(value string, separators Separators) string {
	replacer := strings.NewReplacer(
		separators.Escape, "\\E\\",
		separators.Field, "\\F\\",
		separators.Component, "\\S\\",
		separators.Repetition, "\\R\\",
		separators.Subcomponent, "\\T\\",
	)
	return replacer.Replace(value)
}

func Unescape(value string, separators Separators) string {
	replacer := strings.NewReplacer(
		"\\F\\", separators.Field,
		"\\S\\", separators.Component,
		"\\R\\", separators.Repetition,
		"\\T\\", separators.Subcomponent,
		"\\E\\", separators.Escape,
	)
	return replacer.Replace(value)
}
