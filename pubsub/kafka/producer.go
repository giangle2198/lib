package kafka

import (
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/Shopify/sarama"
)

const MAXIMUMRETRY = 3

type ProducerProvider struct {
	transactionIdGenerator int32

	producersLock sync.Mutex
	producers     []sarama.AsyncProducer

	producerProvider func() sarama.AsyncProducer
}

func NewProducerProvider(brokers []string, kafkaConfigFn func() *sarama.Config) *ProducerProvider {
	provider := &ProducerProvider{}
	provider.producerProvider = func() sarama.AsyncProducer {
		config := kafkaConfigFn()
		suffix := provider.transactionIdGenerator
		// Append transactionIdGenerator to current config.Producer.Transaction.ID to ensure transaction-id uniqueness.
		if config.Producer.Transaction.ID != "" {
			provider.transactionIdGenerator++
			config.Producer.Transaction.ID = config.Producer.Transaction.ID + "-" + fmt.Sprint(suffix)
		}
		producer, err := sarama.NewAsyncProducer(brokers, config)
		if err != nil {
			return nil
		}
		return producer
	}
	return provider
}

func (p *ProducerProvider) Borrow() (producer sarama.AsyncProducer) {
	p.producersLock.Lock()
	defer p.producersLock.Unlock()

	if len(p.producers) == 0 {
		for {
			producer = p.producerProvider()
			if producer != nil {
				return
			}
		}
	}

	index := len(p.producers) - 1
	producer = p.producers[index]
	p.producers = p.producers[:index]
	return
}

func (p *ProducerProvider) Release(producer sarama.AsyncProducer) {
	p.producersLock.Lock()
	defer p.producersLock.Unlock()

	// If released producer is erroneous close it and don't return it to the producer pool.
	if producer.TxnStatus()&sarama.ProducerTxnFlagInError != 0 {
		// Try to close it
		_ = producer.Close()
		return
	}
	p.producers = append(p.producers, producer)
}

func (p *ProducerProvider) Clear() {
	p.producersLock.Lock()
	defer p.producersLock.Unlock()

	for _, producer := range p.producers {
		producer.Close()
	}
	p.producers = p.producers[:0]
}

func (p *ProducerProvider) Send(producer sarama.AsyncProducer, message *sarama.ProducerMessage) error {
	// Start kafka transaction
	err := producer.BeginTxn()
	if err != nil {
		log.Printf("unable to start txn %s\n", err)
		return err
	}

	// Produce some records in transaction
	producer.Input() <- message

	// commit transaction
	err = producer.CommitTxn()
	if err != nil {
		// retry before return error
		err = p.HandleErrorProducer(producer, message)
		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

func (p *ProducerProvider) HandleErrorProducer(producer sarama.AsyncProducer, message *sarama.ProducerMessage) error {
	var (
		retryNum int
		err      error
	)

	for retryNum < MAXIMUMRETRY {
		log.Printf("Producer: unable to commit txn %s\n", err)
		retryNum++
		if producer.TxnStatus()&sarama.ProducerTxnFlagFatalError != 0 {
			// fatal error. need to recreate producer.
			return errors.New(fmt.Sprintf("Producer: %v - producer is in a fatal state, need to recreate it", retryNum))
		}
		// If producer is in abortable state, try to abort current transaction.
		if producer.TxnStatus()&sarama.ProducerTxnFlagAbortableError != 0 {
			err = producer.AbortTxn()
			if err != nil {
				// If an error occured just retry it.
				retryNum--
				log.Printf("Producer: %v - unable to abort transaction: %+v", retryNum, err)
				continue
			}
			return nil
		}
		// if not you can retry
		err = producer.CommitTxn()
		if err != nil {
			log.Printf("Producer: %v - unable to commit txn %s\n", retryNum, err)
			continue
		}
		return nil
	}

	return err
}

type Producer struct {
	producer sarama.AsyncProducer
	topic    string
}

func NewProducer(brokers []string, topic string, kafkaConfigFn func() *sarama.Config) *Producer {
	config := kafkaConfigFn()
	asyncProducer, err := sarama.NewAsyncProducer(brokers, config)
	if err != nil {
		return nil
	}

	producer := &Producer{
		producer: asyncProducer,
		topic:    topic,
	}

	return producer
}

func (p *Producer) Send(key string, data []byte) error {
	// Start kafka transaction
	err := p.producer.BeginTxn()
	if err != nil {
		log.Printf("unable to start txn %s\n", err)
		return err
	}

	// Produce some records in transaction
	p.producer.Input() <- &sarama.ProducerMessage{Topic: p.topic, Key: sarama.ByteEncoder(key), Value: sarama.ByteEncoder(data)}

	// commit transaction
	err = p.producer.CommitTxn()
	if err != nil {
		log.Printf("Producer: unable to commit txn %s\n", err)
		return err
	}
	return nil
}
