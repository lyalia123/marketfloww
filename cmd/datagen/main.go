package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net"
	"time"
)

func main() {
	go startExchange(":40101", "binance")
	go startExchange(":40102", "coinbase")
	go startExchange(":40103", "kucoin")

	select {}
}

func startExchange(addr, exchangeName string) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to start %s: %v", exchangeName, err)
	}
	defer listener.Close()

	log.Printf("%s listening on %s", exchangeName, addr)

	symbols := []string{"BTCUSDT", "ETHUSDT", "DOGEUSDT", "SOLUSDT", "TONUSDT"}
	rand.Seed(time.Now().UnixNano())

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("%s: connection failed: %v", exchangeName, err)
			continue
		}

		go func() {
			defer conn.Close()
			for {
				symbol := symbols[rand.Intn(len(symbols))]
				price := 30000 + rand.Float64()*20000

				msg := map[string]interface{}{
					"symbol":    symbol,
					"price":     price,
					"timestamp": time.Now().UnixNano(),
				}

				jsonMsg, err := json.Marshal(msg)
				if err != nil {
					log.Printf("%s: failed to marshal JSON: %v", exchangeName, err)
					return
				}

				_, err = conn.Write(append(jsonMsg, '\n'))
				if err != nil {
					log.Printf("%s: failed to send data: %v", exchangeName, err)
					return
				}

				time.Sleep(200 * time.Millisecond)
			}
		}()
	}
}
