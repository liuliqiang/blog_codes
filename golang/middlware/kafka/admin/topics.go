package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/IBM/sarama"
)

func ListTopics() (err error) {
	// 创建 Kafka 配置
	config := sarama.NewConfig()
	config.Version = sarama.V3_6_0_0

	// 创建 Admin 客户端
	admin, err := sarama.NewClusterAdmin(strings.Split(brokers, ","), config)
	if err != nil {
		return fmt.Errorf("creating cluster admin: %w", err)
	}
	defer func() {
		if err := admin.Close(); err != nil {
			log.Fatalf("Error closing admin client: %v", err)
		}
	}()

	// 获取 Topic 列表
	topics, err := admin.ListTopics()
	if err != nil {
		return fmt.Errorf("listing topics: %w", err)
	}

	// 打印 Topic 列表
	for name := range topics {
		fmt.Println(name)
	}

	return nil
}
