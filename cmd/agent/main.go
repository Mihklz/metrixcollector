package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Mihklz/metrixcollector/internal/agent"
)

var (
	flagServerAddr string
	flagPollSec    int
	flagReportSec  int
)

func init() {
	flag.StringVar(&flagServerAddr, "a", "localhost:8080", "address of HTTP server")
	flag.IntVar(&flagPollSec, "p", 2, "poll interval in seconds")
	flag.IntVar(&flagReportSec, "r", 10, "report interval in seconds")
}

func main() {
	flag.Parse()

	pollInterval := time.Duration(flagPollSec) * time.Second
	reportInterval := time.Duration(flagReportSec) * time.Second
	serverAddr := "http://" + flagServerAddr

	log.Println("Agent started")
	log.Printf("Poll interval: %v, Report interval: %v, Server: %s", pollInterval, reportInterval, serverAddr)

	tickerPoll := time.NewTicker(pollInterval)
	tickerReport := time.NewTicker(reportInterval)

	var currentMetrics agent.MetricsSet

	for {
		select {
		case <-tickerPoll.C:
			currentMetrics = agent.Collect()
		case <-tickerReport.C:
			go sendMetrics(serverAddr, currentMetrics)
		}
	}
}

func sendMetrics(serverAddr string, metrics agent.MetricsSet) {
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
