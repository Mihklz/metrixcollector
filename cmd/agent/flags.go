package main

import "flag"

var (
	FlagServerAddr string
	FlagReportSec  int
	FlagPollSec    int
)

func parseFlags() {
	flag.StringVar(&FlagServerAddr, "a", "localhost:8080", "server address")
	flag.IntVar(&FlagReportSec, "r", 10, "report interval in seconds")
	flag.IntVar(&FlagPollSec, "p", 2, "poll interval in seconds")
	flag.Parse()
}
