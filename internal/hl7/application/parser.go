package application

import (
	"fmt"
	"github.com/example/hl7v2-message-router/internal/hl7/domain"
	"strings"
	"time"
)

type Parser struct{}

func NewParser() *Parser { return &Parser{} }
func (p *Parser) Parse(raw string) (domain.Message, error) {
	raw = strings.Trim(raw, "\r\n")
	if raw == "" {
		return domain.Message{}, fmt.Errorf("empty HL7 message")
	}
	lines := strings.Split(strings.ReplaceAll(raw, "\n", "\r"), "\r")
	m := domain.Message{Raw: raw, ReceivedAt: time.Now()}
	for _, line := range lines {
		if line == "" {
			continue
		}
		fs := strings.Split(line, "|")
		if len(fs) == 0 {
			continue
		}
		s := domain.Segment{Name: fs[0], Raw: line}
		for _, v := range fs[1:] {
			reps := strings.Split(v, "~")
			s.Fields = append(s.Fields, domain.Field{Value: reps[0], Repetitions: reps, Components: strings.Split(reps[0], "^")})
		}
		m.Segments = append(m.Segments, s)
	}
	if len(m.Segments) == 0 || m.Segments[0].Name != "MSH" {
		return m, fmt.Errorf("MSH segment required")
	}
	msh := m.Segments[0]
	if len(msh.Fields) < 11 {
		return m, fmt.Errorf("MSH requires fields 1..11")
	}
	get := func(i int) string {
		if i >= 0 && i < len(msh.Fields) {
			return msh.Fields[i].Value
		}
		return ""
	}
	m.SendingApplication = get(1)
	m.SendingFacility = get(2)
	m.ReceivingApplication = get(3)
	m.ReceivingFacility = get(4)
	m.ControlID = get(8)
	m.Version = get(10)
	if t := get(7); t != "" {
		m.MessageType = strings.Split(t, "^")[0]
		a := strings.Split(t, "^")
		if len(a) > 1 {
			m.Trigger = a[1]
		}
	}
	if m.ControlID == "" {
		return m, fmt.Errorf("MSH-10 control id required")
	}
	return m, nil
}
