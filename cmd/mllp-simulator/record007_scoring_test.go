package main

import (
	"bufio"
	"net"
	"strings"
	"testing"

	"github.com/example/hl7v2-message-router/internal/mllp"
)

func TestSimulatorACKUsesCurrentControlID(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	go serve(server, "AA", 0)
	reader := bufio.NewReader(client)
	for _, controlID := range []string{"CTRL-007-A", "CTRL-007-B"} {
		message := "MSH|^~\\&|LAB|HOSP|SIM|GW|20260825||ADT^A01|" + controlID + "|P|2.5\r"
		frame := append([]byte{mllp.SB}, []byte(message)...)
		frame = append(frame, mllp.EB, mllp.CR)
		if _, err := client.Write(frame); err != nil {
			t.Fatal(err)
		}
		ack, err := mllp.ReadFrame(reader)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(ack, "MSA|AA|"+controlID) {
			t.Fatalf("ACK for %s = %q", controlID, ack)
		}
	}
}
