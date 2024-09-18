package main

import (
	"automata/client/payeer"
	"log"
	"time"
)

const REQ_COUNT = 600

func main() {
	client := payeer.NewClient(&payeer.Config{})
	start := time.Now()
	for i := range REQ_COUNT {
		rsp, err := client.Orders([]payeer.Pair{payeer.PAIR_BTCUSD})
		if err != nil {
			panic(err)
		}
		if !rsp.Success {
			panic(rsp.Error)
		}
		log.Printf("Fetched: %d/%d", i+1, REQ_COUNT)
	}
	duration := time.Since(start)
	log.Printf("Total requests: %d", REQ_COUNT)
	log.Printf("Total duration: %s", duration)
	log.Printf("Average response time: %.2f ms", float64(duration/REQ_COUNT)/1e6)
}
