package mllp

import (
	"bufio"
	"context"
	"fmt"
	"github.com/example/hl7v2-message-router/internal/hl7/application"
	"io"
	"log/slog"
	"net"
	"time"
)

const SB byte = 0x0b
const EB byte = 0x1c
const CR byte = 0x0d

type Handler interface {
	Handle(context.Context, string) (string, error)
}
type Server struct {
	Addr    string
	Handler Handler
	Logger  *slog.Logger
	ln      net.Listener
}

func (s *Server) Start() error {
	ln, e := net.Listen("tcp", s.Addr)
	if e != nil {
		return e
	}
	s.ln = ln
	go s.accept()
	return nil
}
func (s *Server) accept() {
	for {
		c, e := s.ln.Accept()
		if e != nil {
			return
		}
		go s.conn(c)
	}
}
func (s *Server) conn(c net.Conn) {
	defer c.Close()
	r := bufio.NewReader(c)
	for {
		b, e := r.ReadByte()
		if e != nil {
			return
		}
		if b != SB {
			continue
		}
		payload, e := r.ReadBytes(EB)
		if e != nil {
			return
		}
		if len(payload) < 2 || payload[len(payload)-1] != EB {
			return
		}
		if _, e = r.ReadByte(); e != nil {
			return
		}
		msg := string(payload[:len(payload)-1])
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		ack, he := s.Handler.Handle(ctx, msg)
		cancel()
		if he != nil {
			ack = "AE"
		}
		fmt.Fprintf(c, "%cMSH|^~\\&|ROUTER|GW|||%s||ACK^%s|1|P|2.5\rMSA|%s|%s\r%c%c", SB, time.Now().Format("20060102150405"), application.SafeType(msg), ack, application.SafeControl(msg), EB, CR)
	}
}
func (s *Server) Close() error {
	if s.ln != nil {
		return s.ln.Close()
	}
	return nil
}
func ReadFrame(r io.Reader) (string, error) {
	b := make([]byte, 1)
	if _, e := io.ReadFull(r, b); e != nil {
		return "", e
	}
	if b[0] != SB {
		return "", fmt.Errorf("invalid start block")
	}
	var p []byte
	for {
		if _, e := io.ReadFull(r, b); e != nil {
			return "", e
		}
		if b[0] == EB {
			io.ReadFull(r, b)
			return string(p), nil
		}
		p = append(p, b[0])
	}
}
