package application

import (
	"fmt"
	"strings"

	"github.com/example/hl7v2-message-router/internal/routing/domain"
)

type RouteValidation struct {
	Valid    bool     `json:"valid"`
	Warnings []string `json:"warnings,omitempty"`
	Errors   []string `json:"errors,omitempty"`
}

func ValidateRoute(route domain.Rule, knownTargets map[string]domain.Target) RouteValidation {
	result := RouteValidation{Valid: true}
	if strings.TrimSpace(route.ID) == "" {
		result.Errors = append(result.Errors, "route id is required")
	}
	if strings.TrimSpace(route.Name) == "" {
		result.Warnings = append(result.Warnings, "route name is empty")
	}
	if len(route.TargetIDs) == 0 {
		result.Errors = append(result.Errors, "at least one target is required")
	}
	seen := make(map[string]struct{}, len(route.TargetIDs))
	for _, targetID := range route.TargetIDs {
		if _, duplicate := seen[targetID]; duplicate {
			result.Errors = append(result.Errors, fmt.Sprintf("target %s is duplicated", targetID))
			continue
		}
		seen[targetID] = struct{}{}
		if _, exists := knownTargets[targetID]; !exists {
			result.Errors = append(result.Errors, fmt.Sprintf("target %s does not exist", targetID))
		}
	}
	if route.MessageType == "" && route.Trigger == "" && route.SendingFacility == "" {
		result.Warnings = append(result.Warnings, "route matches all messages")
	}
	if route.Trigger != "" && route.MessageType == "" {
		result.Errors = append(result.Errors, "trigger cannot be used without message type")
	}
	result.Valid = len(result.Errors) == 0
	return result
}

func (s *Store) Validate(id string) (RouteValidation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	route, ok := s.Routes[id]
	if !ok {
		return RouteValidation{}, fmt.Errorf("route %s not found", id)
	}
	return ValidateRoute(route, s.Targets), nil
}

func (s *Store) GetRoute(id string) (domain.Rule, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	route, ok := s.Routes[id]
	return route, ok
}
