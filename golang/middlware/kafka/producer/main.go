package main

import (
	"flag"

	"github.com/rcrowley/go-metrics"
)

// Sarama configuration options
var (
	brokers   = ""
	topic     = ""
	producers = 1
	verbose   = false

	recordsNumber int64 = 1

	recordsRate = metrics.GetOrRegisterMeter("records.rate", nil)
)

func main() {
	flag.StringVar(&brokers, "brokers", "", "Kafka bootstrap brokers to connect to, as a comma separated list")
	flag.StringVar(&topic, "topic", "", "Kafka topics where records will be copied from topics.")
	flag.IntVar(&producers, "producers", 10, "Number of concurrent producers")
	flag.Int64Var(&recordsNumber, "records-number", 10000, "Number of records sent per loop")
	flag.BoolVar(&verbose, "verbose", false, "Sarama logging")
	flag.Parse()

	if len(brokers) == 0 {
		panic("no Kafka bootstrap brokers defined, please set the -brokers flag")
	}

	if len(topic) == 0 {
		panic("no topic given to be consumed, please set the -topic flag")
	}

	// if err := StandaloneSyncMain(); err != nil {
	// 	panic(err)
	// }
	if err := StandaloneAsyncMain(); err != nil {
		panic(err)
	}
	// ConcurrentMain()
}
