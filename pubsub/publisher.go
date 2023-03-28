package pubsub

import (
	"lib/pubsub/kafka"

	"github.com/Shopify/sarama"
	"go.uber.org/zap"
)

type IPublisher interface {
	Send(key string, data []byte) error
}

func NewPublisher(brokers []string, topic, party string) IPublisher {
	var publisher IPublisher

	switch party {
	case "kafka":
		publisher = newKafkaPublisher(brokers, topic)
	default:
		zap.S().Panic("Failed to init publisher")
	}

	return publisher
}

func newKafkaPublisher(brokers []string, topic string) *kafka.Producer {
	producer := kafka.NewProducer(brokers, topic, func() *sarama.Config {
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
	})
	return producer
}
