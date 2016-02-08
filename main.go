package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync/atomic"
)

var (
	paulingLogsAddr       = os.Getenv("MQ_PAULING_LOGS_ADDR")
	logsAddr              = os.Getenv("MQ_LOGS_ADDR")
	listenAddr            = os.Getenv("MQ_LISTEN_ADDR")
	queue           int32 = 1
)

func init() {
	if paulingLogsAddr == "" {
		log.Fatal("MQ_PAULING_LOGS_ADDR not specified")
	}
	if logsAddr == "" {
		logsAddr = ":8002"
	}
}

func main() {
	var err error
	http.HandleFunc("/start", startQueuing)
	http.HandleFunc("/stop", stopQueuing)
	go func() {
		log.Fatal(http.ListenAndServe(listenAddr, http.DefaultServeMux))
	}()

	paulingUDPAddr, err := net.ResolveUDPAddr("udp", paulingLogsAddr)
	if err != nil {
		log.Fatal(err)
	}

	logsUDPAddr, err := net.ResolveUDPAddr("udp", logsAddr)
	if err != nil {
		log.Fatal(err)
	}
	logs, err := net.ListenUDP("udp", logsUDPAddr)
	if err != nil {
		log.Fatal(err)
	}

	conn, err := net.DialUDP("udp", nil, paulingUDPAddr)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Listening for logs on", logsAddr)
	log.Println("Redirecting logs to", paulingLogsAddr)

	var messages [][]byte

	for {
		buff := make([]byte, 65000)
		logs.Read(buff)
		messages = append(messages, buff)

		for i, msg := range messages {
			if atomic.LoadInt32(&queue) == 1 {
				conn.Write(msg)
				messages = append(messages[:i], messages[i+1:]...)
			} else {
				break
			}
		}
	}

}

func startQueuing(w http.ResponseWriter, r *http.Request) {
	atomic.StoreInt32(&queue, 0)
	log.Println("Queuing messages.")
	fmt.Fprint(w, "Queuing messages.")
}

func stopQueuing(w http.ResponseWriter, r *http.Request) {
	atomic.StoreInt32(&queue, 1)
	log.Println("Stopepd queuing messages.")
	fmt.Fprint(w, "Stopped queuing messages.")
}
