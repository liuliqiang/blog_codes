package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/IBM/sarama"
)

func StandaloneSyncMain() (err error) {
	// 创建配置
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll          // 等待所有副本都响应后的响应模式
	config.Producer.Partitioner = sarama.NewRandomPartitioner // 随机分区策略
	config.Producer.Return.Successes = true                   // 成功交付的消息将在success channel返回

	// 使用给定代理地址和配置创建一个生产者
	producer, err := sarama.NewSyncProducer(strings.Split(brokers, ","), config)
	if err != nil {
		return fmt.Errorf("start Sarama producer: %v", err)
	}

	defer func() {
		if err := producer.Close(); err != nil {
			log.Fatalf("Failed to close producer: %v", err)
		}
	}()

	// 捕获退出信号，以便在退出时关闭生产者
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	// 构建要发送的消息
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder("hello-world"),
		Value: sarama.StringEncoder("Hello, Kafka!"),
	}

	// 发送消息
	partition, offset, err := producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("send message: %v", err)
	}

	fmt.Printf("Message %s sent to partition %d at offset %d\n", msg.Key, partition, offset)

	// 等待退出信号
	<-signals
	fmt.Println("Shutting down")
	return nil
}
