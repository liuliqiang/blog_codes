package main

import (
	"flag"
	"log"

	"github.com/IBM/sarama"
)

// Sarama configuration options
var (
	brokers  = ""
	version  = ""
	group    = ""
	topic    = ""
	assignor = ""
	oldest   = true
	verbose  = false
)

func main() {
	flag.StringVar(&brokers, "brokers", "", "Kafka bootstrap brokers to connect to, as a comma separated list")
	flag.StringVar(&group, "group", "", "Kafka consumer group definition")
	flag.StringVar(&version, "version", sarama.DefaultVersion.String(), "Kafka cluster version")
	flag.StringVar(&topic, "topics", "", "Kafka topics to be consumed, as a comma separated list")
	flag.StringVar(&assignor, "assignor", "range", "Consumer group partition assignment strategy (range, roundrobin, sticky)")
	flag.BoolVar(&oldest, "oldest", true, "Kafka consumer consume initial offset from oldest")
	flag.BoolVar(&verbose, "verbose", false, "Sarama logging")
	flag.Parse()

	if err := ListTopics(); err != nil {
		log.Fatalf("Error listing topics: %v", err)
	}
}
