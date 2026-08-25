package application

import (
	"errors"
	"fmt"
	"strings"
	"time"

	hl7 "github.com/example/hl7v2-message-router/internal/hl7/domain"
	"github.com/example/hl7v2-message-router/internal/mapping/domain"
	segment "github.com/example/hl7v2-message-router/internal/segment/application"
)

var ErrCodeTableEntry = errors.New("code table entry not found")

type CodeTable interface {
	Lookup(table, value string) (string, bool)
}

type StaticCodeTables struct{ tables map[string]map[string]string }

func NewStaticCodeTables(values map[string]map[string]string) *StaticCodeTables {
	copyOfTables := make(map[string]map[string]string, len(values))
	for table, values := range values {
		copyOfValues := make(map[string]string, len(values))
		for source, destination := range values {
			copyOfValues[source] = destination
		}
		copyOfTables[table] = copyOfValues
	}
	return &StaticCodeTables{tables: copyOfTables}
}

func (s *StaticCodeTables) Lookup(table, value string) (string, bool) {
	values, ok := s.tables[table]
	if !ok {
		return "", false
	}
	replacement, ok := values[value]
	return replacement, ok
}

type Transformer struct {
	accessor *segment.Accessor
	tables   CodeTable
}

func NewTransformer(tables CodeTable) *Transformer {
	return &Transformer{accessor: segment.NewAccessor(), tables: tables}
}

func (t *Transformer) Transform(message hl7.Message, mapping domain.Mapping) (hl7.Message, []error) {
	clone := cloneMessage(message)
	var failures []error
	for index, operation := range mapping.Operations {
		if err := t.apply(&clone, operation); err != nil {
			failures = append(failures, fmt.Errorf("operation %d destination %s: %w", index+1, operation.Destination, err))
		}
	}
	return clone, failures
}

func (t *Transformer) apply(message *hl7.Message, operation domain.Operation) error {
	value := operation.Default
	if operation.Source != "" {
		resolved, err := t.accessor.Get(*message, operation.Source)
		if err != nil {
			if value == "" {
				return fmt.Errorf("source %s: %w", operation.Source, err)
			}
		} else {
			value = resolved
		}
	}
	if operation.CodeTable != "" {
		if t.tables == nil {
			return fmt.Errorf("code table %s is not configured", operation.CodeTable)
		}
		replacement, ok := t.tables.Lookup(operation.CodeTable, value)
		if !ok {
			return fmt.Errorf("code table %s has no entry for %q: %v", operation.CodeTable, value, ErrCodeTableEntry)
		}
		value = replacement
	}
	if operation.Mask {
		value = maskValue(value)
	}
	if strings.HasPrefix(operation.Destination, "date:") {
		parts := strings.Split(operation.Destination, ":")
		if len(parts) != 4 {
			return fmt.Errorf("date destination must be date:PATH:source-layout:destination-layout")
		}
		parsed, err := time.Parse(parts[2], value)
		if err != nil {
			return fmt.Errorf("parse date at %s: %w", operation.Source, err)
		}
		value = parsed.Format(parts[3])
		operation.Destination = parts[1]
	}
	if operation.Destination == "" {
		return fmt.Errorf("destination is required")
	}
	if err := t.accessor.Set(message, operation.Destination, value); err != nil {
		return fmt.Errorf("set value: %w", err)
	}
	return nil
}

func maskValue(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 4 {
		return strings.Repeat("*", len(value))
	}
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}

func cloneMessage(message hl7.Message) hl7.Message {
	clone := message
	clone.Segments = make([]hl7.Segment, len(message.Segments))
	for i, sourceSegment := range message.Segments {
		clone.Segments[i] = sourceSegment
		clone.Segments[i].Fields = append([]hl7.Field(nil), sourceSegment.Fields...)
	}
	return clone
}
