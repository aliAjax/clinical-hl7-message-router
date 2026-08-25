package application

import (
	"fmt"
	"strings"

	hl7 "github.com/example/hl7v2-message-router/internal/hl7/domain"
	"github.com/example/hl7v2-message-router/internal/segment/domain"
	"github.com/example/hl7v2-message-router/internal/segment/infrastructure"
)

type Accessor struct {
	schemas *infrastructure.SchemaRegistry
}

func NewAccessor() *Accessor { return &Accessor{} }
func NewAccessorWithRegistry(schemas *infrastructure.SchemaRegistry) *Accessor {
	return &Accessor{schemas: schemas}
}

func (a *Accessor) Get(message hl7.Message, value string) (string, error) {
	path, err := domain.ParsePath(value)
	if err != nil {
		return "", err
	}
	segment, err := find(message, path)
	if err != nil {
		return "", err
	}
	index := path.Field - 1
	if index >= len(segment.Fields) {
		return "", fmt.Errorf("%s: field does not exist", path)
	}
	field := segment.Fields[index]
	repetition := field.Repetitions
	if len(repetition) == 0 {
		repetition = []string{field.Value}
	}
	if path.Repetition > len(repetition) {
		return "", fmt.Errorf("%s: repetition does not exist", path)
	}
	value = repetition[path.Repetition-1]
	if path.Component == 0 {
		return value, nil
	}
	components := strings.Split(value, "^")
	if path.Component > len(components) {
		return "", fmt.Errorf("%s: component does not exist", path)
	}
	value = components[path.Component-1]
	if path.Subcomponent == 0 {
		return value, nil
	}
	subcomponents := strings.Split(value, "&")
	if path.Subcomponent > len(subcomponents) {
		return "", fmt.Errorf("%s: subcomponent does not exist", path)
	}
	return subcomponents[path.Subcomponent-1], nil
}

func (a *Accessor) Set(message *hl7.Message, value, replacement string) error {
	path, err := domain.ParsePath(value)
	if err != nil {
		return err
	}
	if a.schemas != nil {
		schema, _ := a.schemas.Get(message.Version)
		if _, found := schema.Lookup(path.Segment); !found {
			return fmt.Errorf("%s: segment is absent from schema %s", path, message.Version)
		}
	}
	segment, err := findMutable(message, path)
	if err != nil {
		return err
	}
	field := &segment.Fields[path.Field-1]
	repetitions := append([]string(nil), field.Repetitions...)
	if len(repetitions) == 0 {
		repetitions = []string{field.Value}
	}
	for len(repetitions) < path.Repetition {
		repetitions = append(repetitions, "")
	}
	if path.Component == 0 {
		repetitions[path.Repetition-1] = replacement
	} else {
		components := strings.Split(repetitions[path.Repetition-1], "^")
		for len(components) < path.Component {
			components = append(components, "")
		}
		if path.Subcomponent == 0 {
			components[path.Component-1] = replacement
		} else {
			subs := strings.Split(components[path.Component-1], "&")
			for len(subs) < path.Subcomponent {
				subs = append(subs, "")
			}
			subs[path.Subcomponent-1] = replacement
			components[path.Component-1] = strings.Join(subs, "&")
		}
		repetitions[path.Repetition-1] = strings.Join(components, "^")
	}
	field.Repetitions = repetitions
	field.Value = repetitions[0]
	field.Components = strings.Split(field.Value, "^")
	return nil
}

func find(message hl7.Message, path domain.Path) (hl7.Segment, error) {
	seen := 0
	for _, segment := range message.Segments {
		if segment.Name == path.Segment {
			seen++
			if seen == path.Occurrence {
				return segment, nil
			}
		}
	}
	return hl7.Segment{}, fmt.Errorf("%s: segment does not exist", path)
}

func findMutable(message *hl7.Message, path domain.Path) (*hl7.Segment, error) {
	seen := 0
	for i := range message.Segments {
		if message.Segments[i].Name == path.Segment {
			seen++
			if seen == path.Occurrence {
				return &message.Segments[i], nil
			}
		}
	}
	return nil, fmt.Errorf("%s: segment does not exist", path)
}
