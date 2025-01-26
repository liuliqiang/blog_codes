package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/IBM/sarama"
)

func StandaloneAsyncMain() (err error) {
	// 创建配置
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true // 启用成功通知
	config.Producer.Return.Errors = true    // 启用错误通知

	// 创建生产者
	producer, err := sarama.NewAsyncProducer(strings.Split(brokers, ","), config)
	if err != nil {
		return fmt.Errorf("start Sarama producer: %v", err)
	}

	// 异步发送消息
	go func() {
		for {
			select {
			case msg := <-producer.Successes():
				fmt.Printf("Produced message to topic %s partition %d at offset %d\n", msg.Topic, msg.Partition, msg.Offset)
			case err := <-producer.Errors():
				log.Printf("Error producing message: %v\n", err)
			}
		}
	}()

	// 捕获退出信号
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	// 从标准输入读取消息，发送到 Kafka
	select {
	case <-signals:
		log.Println("Interrupt signal received, shutting down...")
		if err := producer.Close(); err != nil {
			return fmt.Errorf("close producer: %v", err)
		}
		return
	default:
		msg := &sarama.ProducerMessage{
			Topic: "test-topic",
			Key:   sarama.StringEncoder("hello-world"),
			Value: sarama.StringEncoder("Hello, Kafka!"),
		}
		producer.Input() <- msg
		fmt.Println("Message sent")
	}

	<-signals
	fmt.Println("Shutting down")
	return nil
}
