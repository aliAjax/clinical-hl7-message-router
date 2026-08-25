package application

import (
	hl7app "github.com/example/hl7v2-message-router/internal/hl7/application"
	"testing"
)

func TestGetSetRepeatedComponent(t *testing.T) {
	message, err := hl7app.NewParser().Parse("MSH|^~\\&|LAB|HOSP|EHR|HOSP|20260821150000||ORU^R01|MSG002|P|2.5\rPID|1||12345||DOE^JANE~SMITH^JANE")
	if err != nil {
		t.Fatal(err)
	}
	a := NewAccessor()
	got, err := a.Get(message, "PID-5[2].1")
	if err != nil || got != "SMITH" {
		t.Fatalf("get = %q, %v", got, err)
	}
	if err := a.Set(&message, "PID-5[2].1", "MASKED"); err != nil {
		t.Fatal(err)
	}
	got, _ = a.Get(message, "PID-5[2].1")
	if got != "MASKED" {
		t.Fatalf("after set = %q", got)
	}
}
