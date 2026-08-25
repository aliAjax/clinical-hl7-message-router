package mllp

import (
	"bytes"
	"io"
	"testing"
)

type chunkReader struct{ chunks [][]byte }

func (r *chunkReader) Read(p []byte) (int, error) {
	if len(r.chunks) == 0 {
		return 0, io.EOF
	}
	chunk := r.chunks[0]
	r.chunks = r.chunks[1:]
	n := copy(p, chunk)
	if n < len(chunk) {
		r.chunks = append([][]byte{chunk[n:]}, r.chunks...)
	}
	return n, nil
}

func TestReadFrameHandlesHalfPackets(t *testing.T) {
	r := &chunkReader{chunks: [][]byte{{SB, 'M', 'S'}, {'H', '|', 'a'}, {EB}, {CR}}}
	got, err := ReadFrame(r)
	if err != nil {
		t.Fatal(err)
	}
	if got != "MSH|a" {
		t.Fatalf("frame = %q", got)
	}
}

func TestReadFrameRejectsBadStart(t *testing.T) {
	if _, err := ReadFrame(bytes.NewBufferString("Xbad")); err == nil {
		t.Fatal("expected bad start error")
	}
}
