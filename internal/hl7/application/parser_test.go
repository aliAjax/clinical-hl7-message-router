package application

import "testing"

const sampleADT = "MSH|^~\\&|LAB|HOSP|EHR|HOSP|20260821150000||ADT^A01|MSG001|P|2.5\rEVN|A01|20260821150000\rPID|1||12345^^^HOSP^MR||DOE^JANE\rPV1|1|I"

func TestParserPreservesUnknownSegmentsAndMetadata(t *testing.T) {
	p := NewParser()
	message, err := p.Parse(sampleADT + "\rZXY|custom|unknown")
	if err != nil {
		t.Fatal(err)
	}
	if message.ControlID != "MSG001" {
		t.Fatalf("control id = %q", message.ControlID)
	}
	if message.MessageType != "ADT" || message.Trigger != "A01" {
		t.Fatalf("type = %s^%s", message.MessageType, message.Trigger)
	}
	if message.SendingFacility != "HOSP" {
		t.Fatalf("facility = %q", message.SendingFacility)
	}
	if got := message.Segments[len(message.Segments)-1].Name; got != "ZXY" {
		t.Fatalf("unknown segment = %q", got)
	}
}

func TestParserRejectsMissingHeaderAndControlID(t *testing.T) {
	p := NewParser()
	if _, err := p.Parse("PID|1||123"); err == nil {
		t.Fatal("expected missing MSH error")
	}
	if _, err := p.Parse("MSH|^~\\&|LAB|HOSP|EHR|HOSP|20260821150000||ADT^A01||P|2.5"); err == nil {
		t.Fatal("expected missing control id error")
	}
}

func TestIdempotencyUsesControlID(t *testing.T) {
	p := NewParser()
	left, _ := p.Parse(sampleADT)
	right, _ := p.Parse(sampleADT + "\rNTE|1||changed")
	if left.IdempotencyKey() != right.IdempotencyKey() {
		t.Fatal("same control id must have same idempotency key")
	}
}
