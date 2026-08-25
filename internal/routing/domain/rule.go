package domain

import "fmt"

type RuleState string

const (
	RuleDraft     RuleState = "draft"
	RuleValidated RuleState = "validated"
	RulePublished RuleState = "published"
)

type Target struct {
	ID, Name, Address string
	Healthy           bool
}
type Rule struct {
	ID, Name, MessageType, Trigger, SendingFacility string
	TargetIDs                                       []string
	Parallel                                        bool
	Published                                       bool
	State                                           RuleState
	Version                                         int
}

func (r Rule) Transition(next RuleState) (Rule, error) {
	if next == "" {
		return r, fmt.Errorf("route state is required")
	}
	r.State = next
	r.Published = next == RulePublished
	if r.Published {
		r.Version++
	}
	return r, nil
}

func (r Rule) Matches(msgType, trigger, facility string) bool {
	return (r.MessageType == "" || r.MessageType == msgType) && (r.Trigger == "" || r.Trigger == trigger) && (r.SendingFacility == "" || r.SendingFacility == facility)
}
