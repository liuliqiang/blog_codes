package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/IBM/sarama"
)

func AtMostOnceMain() (err error) {
	// 创建配置
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.AutoCommit.Enable = true // 开启自动提交, 默认也是 true
	config.Version = sarama.V3_6_0_0                 // 设置 Kafka 版本

	consumer, err := sarama.NewConsumer(strings.Split(brokers, ","), config)
	if err != nil {
		return fmt.Errorf("creating consumer: %v", err)
	}
	defer func() {
		if err := consumer.Close(); err != nil {
			fmt.Printf("Error closing consumer: %v\n", err)
		}
	}()

	partitions, err := consumer.Partitions(topic)
	if err != nil {
		return fmt.Errorf("retrieving partitions: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(len(partitions))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	for _, partition := range partitions {
		partitionConsumer, err := consumer.ConsumePartition(topic, partition, sarama.OffsetOldest)
		if err != nil {
			return fmt.Errorf("creating partition consumer: %v", err)
		}

		go func(pc sarama.PartitionConsumer) {
			defer wg.Done()
			defer func() {
				if err := pc.Close(); err != nil {
					fmt.Printf("Error closing partition consumer: %v\n", err)
				}
			}()

			for {
				select {
				case msg := <-pc.Messages():
					fmt.Printf("Partition: %d, Offset: %d, Key: %s, Value: %s\n", msg.Partition, msg.Offset, string(msg.Key), string(msg.Value))
				case err := <-pc.Errors():
					fmt.Printf("Error consuming message: %v\n", err)
				case <-ctx.Done():
					return
				}
			}
		}(partitionConsumer)
	}

	<-signals
	cancel()
	wg.Wait()
	return nil
}
