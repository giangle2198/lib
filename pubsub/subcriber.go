package pubsub

import (
	"context"
	"lib/pubsub/kafka"

	"github.com/Shopify/sarama"
	"go.uber.org/zap"
)

type ISubscriber interface {
	Close() error
	Subscribe(context.Context, []string) error
}

func NewSubscriber(ctx context.Context, brokers []string, party, groupID string, fnMessageHandler func(interface{})) ISubscriber {
	var subscriber ISubscriber

	switch party {
	case "kafka":
		subscriber = newKafkaConsumer(ctx, brokers, groupID, fnMessageHandler)
	default:
		zap.S().Panic("Failed to init subscriber")
	}

	return subscriber
}

func newKafkaConsumer(ctx context.Context, brokers []string, groupID string, fnMessageHandler func(interface{})) *kafka.ConsumerGroup {

	consumerGroup := kafka.NewConsumerGroup(ctx, brokers, groupID, func() *sarama.Config {
		ver, _ := sarama.ParseKafkaVersion("2.6.0")
		config := sarama.NewConfig()
		config.Version = ver
		config.Producer.Idempotent = true
		config.Producer.Return.Errors = false
		config.Producer.RequiredAcks = sarama.WaitForAll
		config.Producer.Partitioner = sarama.NewRoundRobinPartitioner
		config.Producer.Transaction.Retry.Backoff = 10
		config.Producer.Transaction.ID = "txn_producer"
		config.Net.MaxOpenRequests = 1
		return config
	}, fnMessageHandler)

	return consumerGroup
}
