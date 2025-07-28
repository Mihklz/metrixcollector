package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Mihklz/metrixcollector/internal/agent"
)

const (
	pollInterval   = 2 * time.Second
	reportInterval = 10 * time.Second
	serverAddr     = "http://localhost:8080"
)

func main() {
	log.Println("Agent started")

	tickerPoll := time.NewTicker(pollInterval)
	tickerReport := time.NewTicker(reportInterval)

	var currentMetrics agent.MetricsSet

	for {
		select {
		case <-tickerPoll.C:
			currentMetrics = agent.Collect()

		case <-tickerReport.C:
			go sendMetrics(currentMetrics)
		}
	}
}
func sendMetrics(metrics agent.MetricsSet) {
	client := &http.Client{}

	for name, value := range metrics.Gauges {
		url := fmt.Sprintf("%s/update/gauge/%s/%f", serverAddr, name, value)
		req, err := http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			log.Printf("create gauge request error: %v", err)
			continue
		}
		req.Header.Set("Content-Type", "text/plain")

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("send gauge error: %v", err)
			continue
		}
		_ = resp.Body.Close()
	}

	// Отправка counter PollCount
	url := fmt.Sprintf("%s/update/counter/PollCount/%d", serverAddr, metrics.PollCount)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		log.Printf("create counter request error: %v", err)
		return
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("send counter error: %v", err)
		return
	}
	_ = resp.Body.Close()
}
