package mllp

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"
)

func TestReadFrameRequiresCarriageReturn(t *testing.T) {
	frame := append([]byte{SB}, []byte("MSH|bad-terminator")...)
	frame = append(frame, EB, '\n')
	if _, err := ReadFrame(bytes.NewReader(frame)); err == nil {
		t.Fatal("frame with non-CR terminator was accepted")
	}
}

type cancellationHandler struct {
	started chan struct{}
	done    chan error
}

func (h *cancellationHandler) Handle(ctx context.Context, _ string) (string, error) {
	close(h.started)
	<-ctx.Done()
	h.done <- ctx.Err()
	return "", ctx.Err()
}

func TestConnectionCancellationStopsHandler(t *testing.T) {
	handler := &cancellationHandler{started: make(chan struct{}), done: make(chan error, 1)}
	server := &Server{Addr: "127.0.0.1:0", Handler: handler}
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	connection, err := net.Dial("tcp", server.ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	frame := append([]byte{SB}, []byte("MSH|^~\\&|LAB|HOSP|GW|SITE|20260825||ADT^A01|CTRL-007|P|2.5\r")...)
	frame = append(frame, EB, CR)
	if _, err := connection.Write(frame); err != nil {
		t.Fatal(err)
	}
	select {
	case <-handler.started:
	case <-time.After(time.Second):
		t.Fatal("handler did not start")
	}
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-handler.done:
		if err != context.Canceled {
			t.Fatalf("handler stopped with %v", err)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("handler remained blocked after server close")
	}
}
