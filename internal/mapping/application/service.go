package application

import (
	hl7 "github.com/example/hl7v2-message-router/internal/hl7/domain"
	"github.com/example/hl7v2-message-router/internal/mapping/domain"
	"github.com/example/hl7v2-message-router/internal/mapping/infrastructure"
)

type Service struct {
	store       *infrastructure.Store
	transformer *Transformer
}

func New() *Service { return NewWithDependencies(infrastructure.NewStore(), NewTransformer(nil)) }
func NewWithDependencies(store *infrastructure.Store, transformer *Transformer) *Service {
	return &Service{store: store, transformer: transformer}
}
func (s *Service) Put(m domain.Mapping) { s.store.Put(m) }
func (s *Service) Apply(m hl7.Message, id string) (hl7.Message, []error) {
	mapping, err := s.store.Get(id)
	if err != nil {
		return m, []error{err}
	}
	transformed, failures := s.transformer.Transform(m, mapping)
	if len(failures) > 1 {
		failures = failures[:1]
	}
	return transformed, failures
}
