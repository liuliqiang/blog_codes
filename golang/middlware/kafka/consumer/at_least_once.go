package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/IBM/sarama"
)

func AtLeastOnceMain() error {
	if verbose {
		sarama.Logger = log.New(os.Stdout, "[sarama] ", log.LstdFlags)
	}

	// 创建配置
	config := sarama.NewConfig()
	config.Version = sarama.V3_6_0_0
	config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRoundRobin(),
	}
	config.Consumer.Offsets.Initial = sarama.OffsetNewest

	consumer := CustomCommitConsumer{
		ready: make(chan bool),
	}

	client, err := sarama.NewConsumerGroup(strings.Split(brokers, ","), group, config)
	if err != nil {
		return fmt.Errorf("creating consumer group client: %+w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			// 如果发生 rebalance 的时候，Consume 就会返回，此时我们需要重新创建一个新的 Consumer
			if err := client.Consume(ctx, strings.Split(topic, ","), &consumer); err != nil {
				switch {
				case errors.Is(err, sarama.ErrClosedConsumerGroup):
					fmt.Printf("Consumer group has been closed: %v\n", err)
					return
				case errors.Is(err, context.Canceled):
					fmt.Printf("Consumer group has been canceled: %v\n", err)
					return
				default:
					log.Panicf("Error from consumer: %v", err)
				}
			}
			log.Println("Consumer has been return")
			consumer.ready = make(chan bool)
		}
	}()

	<-consumer.ready // wait for the consumer have been Setuped.
	log.Println("Sarama consumer up and running!...")

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	<-sigterm
	log.Println("terminating: via signal")
	cancel()
	wg.Wait()
	if err = client.Close(); err != nil {
		return fmt.Errorf("closing client: %+w", err)
	}
	return nil
}

type CustomCommitConsumer struct {
	ready chan bool
}

func (c *CustomCommitConsumer) Setup(sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	close(c.ready)
	return nil
}

func (c *CustomCommitConsumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (c *CustomCommitConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				log.Printf("message channel was closed")
				return nil
			}
			log.Printf("Message claimed: value = %s, timestamp = %v, topic = %s", string(message.Value), message.Timestamp, message.Topic)
			session.MarkMessage(message, "") // 主动提交 Offset
		case <-session.Context().Done():
			return nil
		}
	}
}
