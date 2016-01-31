package main

import (
	"io"
	"log"
	"net"
	"net/rpc"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

var (
	paulingPort     = os.Getenv("PAULING_RPC_PORT")
	paulingLogsPort = os.Getenv("PAULING_LOGS_PORT")
	logsPort        = os.Getenv("LOGS_PORT")
	client          *rpc.Client
	mu              = new(sync.RWMutex)
	messages        = make(chan []byte)
)

func init() {
	if paulingPort == "" {
		log.Fatal("PAULING_RPC_PORT not specified")
	}
	if paulingLogsPort == "" {
		log.Fatal("PAULING_LOGS_PORT not specified")
	}
	if logsPort == "" {
		logsPort = "8002"
	}
}

func main() {
	var err error

	client, err = rpc.DialHTTP("tcp", "localhost:"+paulingPort)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected to Pauling on localhost:" + paulingPort)

	paulingAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:"+paulingLogsPort)
	if err != nil {
		log.Fatal(err)
	}

	logsAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:"+logsPort)
	if err != nil {
		log.Fatal(err)
	}
	logs, err := net.ListenUDP("udp", logsAddr)
	if err != nil {
		log.Fatal(err)
	}

	conn, err := net.DialUDP("udp", nil, paulingAddr)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Listening for logs on", logsAddr)
	log.Println("Redirecting logs to", paulingAddr)

	var connected int32 = 1
	var messages [][]byte

	go func() {
		for {
			buff := make([]byte, 65000)
			logs.Read(buff)
			messages = append(messages, buff)

			for i, msg := range messages {
				if atomic.LoadInt32(&connected) == 1 {
					conn.Write(msg)
					messages = append(messages[:i], messages[i+1:]...)
				} else {
					break
				}
			}
		}
	}()

	for {
		err := client.Call("Pauling.Ping", struct{}{}, &struct{}{})
		if err != nil {
			log.Println("Disonnected from Pauling, queuing messages.")
			for {
				if !isNetworkError(err) {
					log.Fatal(err)
				}

				time.Sleep(1 * time.Second)
				mu.Lock()
				client, err = rpc.DialHTTP("tcp", "localhost:"+paulingPort)
				mu.Unlock()
				if err == nil {
					log.Println("Connected to Pauling.")
					atomic.StoreInt32(&connected, 1)
					break
				}
			}
		}
	}
}

func isNetworkError(err error) bool {
	_, ok := err.(*net.OpError)
	return ok || err == io.ErrUnexpectedEOF || err == rpc.ErrShutdown

}
