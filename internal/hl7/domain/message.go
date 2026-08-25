package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type Field struct {
	Value       string
	Repetitions []string
	Components  []string
}
type Segment struct {
	Name   string
	Fields []Field
	Raw    string
}
type Message struct {
	ID, ControlID, MessageType, Trigger, SendingApplication, SendingFacility, ReceivingApplication, ReceivingFacility string
	Version                                                                                                           string
	Segments                                                                                                          []Segment
	Raw                                                                                                               string
	ReceivedAt                                                                                                        time.Time
}

func (m Message) IdempotencyKey() string {
	if m.ControlID != "" {
		return m.ControlID
	}
	h := sha256.Sum256([]byte(m.Raw))
	return hex.EncodeToString(h[:])
}
func (m Message) Summary() map[string]string {
	return map[string]string{"id": m.ID, "control_id": m.ControlID, "type": m.MessageType, "trigger": m.Trigger, "version": m.Version}
}
func (m Message) Get(path string) (string, error) {
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid field path %q", path)
	}
	seg := parts[0]
	idx := 0
	if len(parts) > 1 {
		_, _ = fmt.Sscanf(parts[1], "%d", &idx)
	}
	for _, s := range m.Segments {
		if s.Name == seg {
			if idx < 0 || idx >= len(s.Fields) {
				return "", fmt.Errorf("%s.%d out of range", seg, idx)
			}
			return s.Fields[idx].Value, nil
		}
	}
	return "", fmt.Errorf("segment %s not found", seg)
}
