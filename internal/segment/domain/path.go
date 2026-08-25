package domain

import (
	"fmt"
	"strconv"
	"strings"
)

// Path identifies a field, repetition, component, and subcomponent using
// one-based indexes as used by HL7 documentation (for example PID-5[1].1).
type Path struct {
	Segment      string
	Occurrence   int
	Field        int
	Repetition   int
	Component    int
	Subcomponent int
}

func ParsePath(value string) (Path, error) {
	var out Path
	value = strings.TrimSpace(value)
	if value == "" {
		return out, fmt.Errorf("empty segment path")
	}
	dash := strings.IndexByte(value, '-')
	if dash < 1 {
		return out, fmt.Errorf("path %q must contain segment and field", value)
	}
	segmentPart := value[:dash]
	out.Segment = segmentPart
	out.Occurrence = 1
	if open := strings.IndexByte(segmentPart, '['); open >= 0 {
		close := strings.IndexByte(segmentPart[open:], ']')
		if close < 0 {
			return out, fmt.Errorf("unclosed segment occurrence in %q", value)
		}
		var err error
		out.Occurrence, err = positive(segmentPart[open+1 : open+close])
		if err != nil {
			return out, fmt.Errorf("segment occurrence: %w", err)
		}
		out.Segment = segmentPart[:open]
	}
	if len(out.Segment) != 3 {
		return out, fmt.Errorf("segment name %q must be three characters", out.Segment)
	}
	rest := value[dash+1:]
	fieldPart, rest := take(rest, '.')
	if open := strings.IndexByte(fieldPart, '['); open >= 0 {
		close := strings.IndexByte(fieldPart[open:], ']')
		if close < 0 {
			return out, fmt.Errorf("unclosed repetition in %q", value)
		}
		var err error
		out.Repetition, err = positive(fieldPart[open+1 : open+close])
		if err != nil {
			return out, fmt.Errorf("field repetition: %w", err)
		}
		fieldPart = fieldPart[:open]
	} else {
		out.Repetition = 1
	}
	var err error
	out.Field, err = positive(fieldPart)
	if err != nil {
		return out, fmt.Errorf("field index: %w", err)
	}
	if rest == "" {
		return out, nil
	}
	componentPart, remainder := take(rest, '.')
	out.Component, err = positive(componentPart)
	if err != nil {
		return out, fmt.Errorf("component index: %w", err)
	}
	if remainder != "" {
		out.Subcomponent, err = positive(remainder)
		if err != nil {
			return out, fmt.Errorf("subcomponent index: %w", err)
		}
	}
	return out, nil
}

func (p Path) String() string {
	segment := p.Segment
	if p.Occurrence > 1 {
		segment += fmt.Sprintf("[%d]", p.Occurrence)
	}
	field := fmt.Sprintf("%s-%d", segment, p.Field)
	if p.Repetition > 1 {
		field += fmt.Sprintf("[%d]", p.Repetition)
	}
	if p.Component > 0 {
		field += fmt.Sprintf(".%d", p.Component)
	}
	if p.Subcomponent > 0 {
		field += fmt.Sprintf(".%d", p.Subcomponent)
	}
	return field
}

func positive(value string) (int, error) {
	v, err := strconv.Atoi(value)
	if err != nil || v < 1 {
		return 0, fmt.Errorf("%q is not a positive integer", value)
	}
	return v, nil
}

func take(value string, separator byte) (string, string) {
	if i := strings.IndexByte(value, separator); i >= 0 {
		return value[:i], value[i+1:]
	}
	return value, ""
}
