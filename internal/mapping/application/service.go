package application

import (
	"fmt"
	hl7 "github.com/example/hl7v2-message-router/internal/hl7/domain"
	"github.com/example/hl7v2-message-router/internal/mapping/domain"
	"strings"
	"time"
)

type Service struct{ mappings map[string]domain.Mapping }

func New() *Service                     { return &Service{mappings: map[string]domain.Mapping{}} }
func (s *Service) Put(m domain.Mapping) { s.mappings[m.ID] = m }
func (s *Service) Apply(m hl7.Message, id string) (hl7.Message, []error) {
	mp, ok := s.mappings[id]
	if !ok {
		return m, []error{fmt.Errorf("mapping %s not found", id)}
	}
	errs := []error{}
	for _, op := range mp.Operations {
		v, e := m.Get(op.Source)
		if e != nil {
			if op.Default != "" {
				v = op.Default
			} else {
				errs = append(errs, e)
				continue
			}
		}
		if op.CodeTable != "" {
			v = strings.ToUpper(v)
		}
		if op.Mask {
			v = "[REDACTED]"
		}
		_ = v
	}
	_ = time.Now()
	return m, errs
}
