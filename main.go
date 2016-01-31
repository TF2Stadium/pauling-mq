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
	paulingAddr     = os.Getenv("PAULING_RPC_ADDR")
	paulingLogsAddr = os.Getenv("PAULING_LOGS_ADDR")
	logsAddr        = os.Getenv("LOGS_ADDR")
	client          *rpc.Client
	mu              = new(sync.RWMutex)
	messages        = make(chan []byte)
)

func init() {
	if paulingAddr == "" {
		log.Fatal("PAULING_RPC_ADDR not specified")
	}
	if paulingLogsAddr == "" {
		log.Fatal("PAULING_LOGS_ADDR not specified")
	}
	if logsAddr == "" {
		logsAddr = "8002"
	}
}

func main() {
	var err error

	client, err = rpc.DialHTTP("tcp", paulingAddr)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected to Pauling on" + paulingAddr)

	paulingUDPAddr, err := net.ResolveUDPAddr("udp", paulingAddr)
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
