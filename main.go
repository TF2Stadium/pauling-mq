package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"sync/atomic"

	"github.com/kelseyhightower/envconfig"
)

type c struct {
	PaulingRedirectAddr string `envconfig:"PAULING_REDIRECT_ADDR" required:"true"`
	LogsAddr            string `envconfig:"LOGS_ADDR" default:"0.0.0.0:8002"`
	ListenAddr          string `envconfig:"LISTEN_ADDR" required:"true"`
}

var queue int32

func main() {
	var config c

	err := envconfig.Process("MQ", &config)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/start", startQueuing)
	http.HandleFunc("/stop", stopQueuing)
	go func() {
		log.Fatal(http.ListenAndServe(config.ListenAddr, http.DefaultServeMux))
	}()

	paulingUDPAddr, err := net.ResolveUDPAddr("udp", config.PaulingRedirectAddr)
	if err != nil {
		log.Fatal(err)
	}

	logsUDPAddr, err := net.ResolveUDPAddr("udp", config.LogsAddr)
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

	log.Println("Listening for logs on", config.LogsAddr)
	log.Println("Redirecting logs to", config.PaulingRedirectAddr)

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
