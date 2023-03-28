package kafka

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/Shopify/sarama"
	"go.uber.org/zap"
)

type ConsumerGroup struct {
	errorChan   chan error
	brokers     []string
	groupID     string
	kafkaConfig *sarama.Config
	consumer    Consumer
	keepRunning chan bool
	cg          sarama.ConsumerGroup
}

// -brokers="127.0.0.1:9092" -topics="sarama" -group="example"
func NewConsumerGroup(ctx context.Context, brokers []string, groupID string, kafkaConfigFn func() *sarama.Config, fnMessageHandler interface{}) *ConsumerGroup {

	sarama.Logger = log.New(os.Stdout, "[comsumer]", log.LstdFlags)

	// vrs, err := sarama.ParseKafkaVersion(version)
	// if err != nil {
	// 	return nil, errors.New("Init kafka consumer group failed!")
	// }

	// config := sarama.NewConfig()
	// config.Version = vrs

	// switch strategy {
	// case "sticky":
	// 	config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategySticky}
	// case "roundrobin":
	// 	config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRoundRobin}
	// case "range":
	// 	config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRange}
	// default:
	// 	return nil, errors.New(fmt.Sprintf("Unrecognized consumer group partition assignor: %s", strategy))
	// }

	// if oldest {
	// 	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	// }
	config := kafkaConfigFn()

	consumer := Consumer{
		ready:   make(chan bool),
		handler: fnMessageHandler.(MessageHanlder),
	}

	cg := &ConsumerGroup{
		errorChan:   make(chan error),
		consumer:    consumer,
		kafkaConfig: config,
		brokers:     brokers,
		groupID:     groupID,
		keepRunning: make(chan bool),
	}

	cg.keepRunning <- true
	client, err := sarama.NewConsumerGroup(cg.brokers, cg.groupID, cg.kafkaConfig)
	if err != nil {
		zap.S().Panic(errors.New(fmt.Sprintf("Error creating consumer group client: %v", err)))
		return nil
	}

	cg.cg = client

	return cg
}

func (cg *ConsumerGroup) Subscribe(ctx context.Context, topics []string) error {

	go func() {
		for {
			if ok := <-cg.keepRunning; !ok {
				return
			}

			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims
			if err := cg.cg.Consume(ctx, topics, &cg.consumer); err != nil {
				cg.errorChan <- errors.New(fmt.Sprintf("Error from consumer: %v", err))
				return
			}

			if ctx.Err() != nil {
				return
			}

			cg.consumer.ready = make(chan bool)
		}
	}()

	<-cg.consumer.ready // Await till the consumer has been set up
	log.Println("Sarama consumer up and running!...")

	return nil
}

func (cg *ConsumerGroup) Close() error {
	cg.keepRunning <- false

	if err := cg.cg.Close(); err != nil {
		return err
	}
	return nil
}

type MessageHanlder func(*sarama.ConsumerMessage)

// Consumer represents a Sarama consumer group consumer
type Consumer struct {
	ready   chan bool
	handler MessageHanlder
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (consumer *Consumer) Setup(sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	close(consumer.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (consumer *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (consumer *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			log.Printf("Message claimed: value = %s, timestamp = %v, topic = %s", string(message.Value), message.Timestamp, message.Topic)
			session.MarkMessage(message, "")
			consumer.handler(message)

		case <-session.Context().Done():
			return nil
		}
	}
}
