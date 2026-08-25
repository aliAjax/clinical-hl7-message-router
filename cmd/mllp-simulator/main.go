package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/example/hl7v2-message-router/internal/hl7/application"
	"github.com/example/hl7v2-message-router/internal/mllp"
)

func main() {
	address := flag.String("listen", ":3575", "MLLP listen address")
	ackCode := flag.String("ack", "AA", "ACK code: AA, AE or AR")
	delay := flag.Duration("delay", 0, "delay before ACK")
	flag.Parse()
	listener, err := net.Listen("tcp", *address)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("MLLP simulator listening on %s", *address)
	for {
		connection, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go serve(connection, strings.ToUpper(*ackCode), *delay)
	}
}

func serve(connection net.Conn, code string, delay time.Duration) {
	defer connection.Close()
	reader := bufio.NewReader(connection)
	for {
		message, err := mllp.ReadFrame(reader)
		if err != nil {
			return
		}
		if delay > 0 {
			time.Sleep(delay)
		}
		controlID := application.SafeControl(message)
		messageType := application.SafeType(message)
		ack := fmt.Sprintf("MSH|^~\\&|SIMULATOR|TARGET|ROUTER|GW|%s||ACK^%s|ACK-%s|P|2.5\rMSA|%s|%s\r", time.Now().Format("20060102150405"), messageType, controlID, code, controlID)
		_, _ = fmt.Fprintf(connection, "%c%s%c%c", mllp.SB, ack, mllp.EB, mllp.CR)
	}
}
