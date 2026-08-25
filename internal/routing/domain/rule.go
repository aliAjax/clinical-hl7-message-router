package domain

type Target struct {
	ID, Name, Address string
	Healthy           bool
}
type Rule struct {
	ID, Name, MessageType, Trigger, SendingFacility string
	TargetIDs                                       []string
	Parallel                                        bool
	Published                                       bool
	Version                                         int
}

func (r Rule) Matches(msgType, trigger, facility string) bool {
	return (r.MessageType == "" || r.MessageType == msgType) && (r.Trigger == "" || r.Trigger == trigger) && (r.SendingFacility == "" || r.SendingFacility == facility)
}
