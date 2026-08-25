package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/example/hl7v2-message-router/internal/mllp"
)

func main() {
	address := flag.String("address", "127.0.0.1:2575", "MLLP server address")
	partial := flag.Bool("partial", false, "send the frame as several TCP writes")
	timeout := flag.Duration("timeout", 3*time.Second, "connection and read timeout")
	flag.Parse()
	message := "MSH|^~\\&|PROBE|HOSP|ROUTER|GW|20260821150000||ADT^A01|MLLP-PROBE-001|P|2.5\rPID|1||12345||DOE^JANE\r"
	connection, err := net.DialTimeout("tcp", *address, *timeout)
	if err != nil {
		log.Fatal(err)
	}
	defer connection.Close()
	_ = connection.SetDeadline(time.Now().Add(*timeout))
	frame := append([]byte{mllp.SB}, []byte(message)...)
	frame = append(frame, mllp.EB, mllp.CR)
	if err := writeFrame(connection, frame, *partial); err != nil {
		log.Fatal(err)
	}
	ack, err := mllp.ReadFrame(connection)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintln(os.Stdout, ack)
}

func writeFrame(connection io.Writer, frame []byte, partial bool) error {
	if partial {
		points := []int{3, 11, len(frame) - 2, len(frame)}
		start := 0
		for _, end := range points {
			if _, err := connection.Write(frame[start:end]); err != nil {
				if end == len(frame) {
					return nil
				}
				return err
			}
			start = end
			time.Sleep(20 * time.Millisecond)
		}
	} else if _, err := connection.Write(frame); err != nil {
		return err
	}
	return nil
}
