package main

import (
	"errors"
	"testing"
)

var errFinalFrameWrite = errors.New("final frame write failed")

type finalWriteFailure struct{ writes int }

func (w *finalWriteFailure) Write(data []byte) (int, error) {
	w.writes++
	if w.writes == 4 {
		return 0, errFinalFrameWrite
	}
	return len(data), nil
}

func TestProbePartialFramePreservesBoundaries(t *testing.T) {
	writer := &finalWriteFailure{}
	frame := []byte("0123456789AB")
	if err := writeFrame(writer, frame, true); !errors.Is(err, errFinalFrameWrite) {
		t.Fatalf("write error = %v", err)
	}
	if writer.writes != 4 {
		t.Fatalf("writes = %d", writer.writes)
	}
}
